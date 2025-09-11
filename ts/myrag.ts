// Converted from myrag.py
// This script implements a simple RAG (Retrieval-Augmented Generation) chatbot using Node.js and TypeScript.
// You will need to install dependencies: axios (for HTTP requests) and readline-sync (for CLI input)

import fs from 'fs';
import readlineSync from 'readline-sync';
import ollama from 'ollama';

// Load the dataset
const dataset: string[] = fs.readFileSync('./cat-facts.txt', 'utf-8').split('\n').filter(Boolean);
console.log(`Loaded ${dataset.length} entries`);

// Implement the retrieval system
const EMBEDDING_MODEL = 'hf.co/CompendiumLabs/bge-base-en-v1.5-gguf';
const LANGUAGE_MODEL = 'hf.co/bartowski/Llama-3.2-1B-Instruct-GGUF';

// Each element in the VECTOR_DB will be a tuple [chunk, embedding]
// The embedding is a number[]
const VECTOR_DB: Array<[string, number[]]> = [];


async function getEmbedding(text: string): Promise<number[]> {
  // Use the ollama npm package to get embeddings
  const response = await ollama.embeddings({ model: EMBEDDING_MODEL, prompt: text });
  return response.embedding;
}

async function addChunkToDatabase(chunk: string) {
  const embedding = await getEmbedding(chunk);
  VECTOR_DB.push([chunk, embedding]);
}

function cosineSimilarity(a: number[], b: number[]): number {
  const dotProduct = a.reduce((sum, x, i) => sum + x * b[i], 0);
  const normA = Math.sqrt(a.reduce((sum, x) => sum + x * x, 0));
  const normB = Math.sqrt(b.reduce((sum, x) => sum + x * x, 0));
  return dotProduct / (normA * normB);
}

async function retrieve(query: string, topN = 3): Promise<Array<[string, number]>> {
  const queryEmbedding = await getEmbedding(query);
  const similarities: Array<[string, number]> = VECTOR_DB.map(([chunk, embedding]) => [chunk, cosineSimilarity(queryEmbedding, embedding)]);
  similarities.sort((a, b) => b[1] - a[1]);
  return similarities.slice(0, topN);
}

// Main chatbot loop
async function main() {
  // Build the vector DB
  for (let i = 0; i < dataset.length; i++) {
    await addChunkToDatabase(dataset[i]);
    // console.log(`Added chunk ${i + 1}/${dataset.length} to the database`);
  }

  // Support both piped/redirected stdin and interactive input
  let inputQuery = '';
  if (!process.stdin.isTTY) {
    // Read from stdin (piped or redirected)
    inputQuery = await new Promise<string>((resolve) => {
      let data = '';
      process.stdin.setEncoding('utf-8');
      process.stdin.on('data', chunk => data += chunk);
      process.stdin.on('end', () => resolve(data.trim()));
    });
    if (!inputQuery) {
      // fallback to interactive if nothing was piped
      inputQuery = readlineSync.question('Ask me a question: ');
    }
  } else {
    // Interactive
    inputQuery = readlineSync.question('Ask me a question: ');
  }
  const retrievedKnowledge = await retrieve(inputQuery);

  console.log('Retrieved knowledge:');
  for (const [chunk, similarity] of retrievedKnowledge) {
    console.log(` - (similarity: ${similarity.toFixed(2)}) ${chunk}`);
  }

  const nl = '\n';
  const instructionPrompt = `You are a helpful chatbot.\nUse only the following pieces of context to answer the question. Don't make up any new information:\n${retrievedKnowledge.map(([chunk]) => ` - ${chunk}`).join(nl)}`;

  // console.log(instructionPrompt)
  // Use the ollama npm package to chat
  const response = await ollama.chat({
    model: LANGUAGE_MODEL,
    messages: [
      { role: 'system', content: instructionPrompt },
      { role: 'user', content: inputQuery },
    ],
    options: { temperature: 0.1 },
    stream: false,
  });
  console.log('Chatbot response:');
  console.log(response.message.content);
}

main().catch(console.error);
