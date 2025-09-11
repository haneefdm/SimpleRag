package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"sort"
	"strings"
)

type EmbeddingResponse struct {
	Embedding []float64 `json:"embedding"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

type ChatResponse struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
}

const EMBEDDING_MODEL = "hf.co/CompendiumLabs/bge-base-en-v1.5-gguf"
const LANGUAGE_MODEL = "hf.co/bartowski/Llama-3.2-1B-Instruct-GGUF"

func loadDataset(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var dataset []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			dataset = append(dataset, line)
		}
	}
	return dataset, scanner.Err()
}

func getEmbedding(text string) ([]float64, error) {
	body, _ := json.Marshal(map[string]interface{}{
		"model":  EMBEDDING_MODEL,
		"prompt": text,
	})
	resp, err := http.Post("http://localhost:11434/api/embeddings", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Embedding, nil
}

type VectorDBEntry struct {
	Chunk     string
	Embedding []float64
}

func cosineSimilarity(a, b []float64) float64 {
	dot, normA, normB := 0.0, 0.0, 0.0
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

func retrieve(query string, db []VectorDBEntry, topN int) ([][2]interface{}, error) {
	queryEmbedding, err := getEmbedding(query)
	if err != nil {
		return nil, err
	}
	type simEntry struct {
		Chunk      string
		Similarity float64
	}
	var sims []simEntry
	for _, entry := range db {
		sim := cosineSimilarity(queryEmbedding, entry.Embedding)
		sims = append(sims, simEntry{entry.Chunk, sim})
	}
	sort.Slice(sims, func(i, j int) bool { return sims[i].Similarity > sims[j].Similarity })
	var top [][2]interface{}
	for i := 0; i < topN && i < len(sims); i++ {
		top = append(top, [2]interface{}{sims[i].Chunk, sims[i].Similarity})
	}
	return top, nil
}

func chat(inputQuery string, retrieved [][2]interface{}) (string, error) {
	nl := "\n"
	var contextLines []string
	for _, pair := range retrieved {
		contextLines = append(contextLines, fmt.Sprintf(" - %s", pair[0].(string)))
	}
	instructionPrompt := fmt.Sprintf("You are a helpful chatbot.\nUse only the following pieces of context to answer the question. Don't make up any new information:\n%s", strings.Join(contextLines, nl))

	chatReq := ChatRequest{
		Model: LANGUAGE_MODEL,
		Messages: []ChatMessage{
			{Role: "system", Content: instructionPrompt},
			{Role: "user", Content: inputQuery},
		},
		Stream: false,
	}
	body, _ := json.Marshal(chatReq)
	resp, err := http.Post("http://localhost:11434/api/chat", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", err
	}
	return chatResp.Message.Content, nil
}

func main() {
	dataset, err := loadDataset("cat-facts.txt")
	if err != nil {
		fmt.Println("Error loading dataset:", err)
		return
	}
	fmt.Printf("Loaded %d entries\n", len(dataset))

	var vectorDB []VectorDBEntry
	for i, chunk := range dataset {
		emb, err := getEmbedding(chunk)
		if err != nil {
			fmt.Printf("Error embedding chunk %d: %v\n", i+1, err)
			return
		}
		vectorDB = append(vectorDB, VectorDBEntry{chunk, emb})
		// fmt.Printf("Added chunk %d/%d to the database\n", i+1, len(dataset))
	}

	var inputQuery string
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		// Piped input
		inBytes, _ := io.ReadAll(os.Stdin)
		inputQuery = strings.TrimSpace(string(inBytes))
		if inputQuery == "" {
			fmt.Print("Ask me a question: ")
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				inputQuery = scanner.Text()
			}
		}
	} else {
		fmt.Print("Ask me a question: ")
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			inputQuery = scanner.Text()
		}
	}

	retrieved, err := retrieve(inputQuery, vectorDB, 3)
	if err != nil {
		fmt.Println("Error retrieving knowledge:", err)
		return
	}
	fmt.Println("Retrieved knowledge:")
	for _, pair := range retrieved {
		fmt.Printf(" - (similarity: %.2f) %s\n", pair[1].(float64), pair[0].(string))
	}

	response, err := chat(inputQuery, retrieved)
	if err != nil {
		fmt.Println("Error from chatbot:", err)
		return
	}
	fmt.Println("Chatbot response:")
	fmt.Println(response)
}
