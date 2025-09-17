This is a simple exercise on learning about RAG. It has no useful purpose other than sharing with a few people.

I took a RAG tutorial from [Hugging Face article](https://huggingface.co/blog/ngxson/make-your-own-rag)

It works of course. But I wanted to see if I can use another languages like TS and Golang. It works after a bit of massaging and GH Copilot was a LOT of help. Even the python script from HuggingFace had one minor issue. Go was the easiest to translate to, that was surprising.

All three versions gave wildly different results for the same prompt. I found out why and set the temperature to 0.1 for all versions. This made it much more stable and gave similar results as expected.

As I said before, this is probably absolutely useless for anybody else but maybe it is your first RAG.

Make sure you got all the prerequisites satisfied from [Hugging Face article](https://huggingface.co/blog/ngxson/make-your-own-rag). At the root directory, try one of these demos (or all of them)
1. Python.
```
cat prompt.txt | python3 myrag.py
```
1. TypeScript: You will need [NodeJS](https://nodejs.org/en/download/) installed and also install TypeScript `npm install -g typescript`
```
npm install
npm run compile
cat prompt.txt | npm run start
```
1. Golang/go: You will need [Go Installed](https://go.dev/doc/install)
```
cd go
go build
cat ../prompt.txt | ./ragchat
```

# Observations
1. Temperature has a HUGE impact on results
2. While the prompt+context did ask the model to NOT look at other sources, it seemed like it was getting info from other places. If you ask "How many planets are there in the solar system", you will get an answer while it confesses that it does not have any info on that. We should try setting temperature to 0
3. Try timing these commands. Even though all three have Ollama REST APIs to deal with, the Go version is still roughly 3 times faster
4. Play with your own prompts and hopefully your observations are similar to mine.
