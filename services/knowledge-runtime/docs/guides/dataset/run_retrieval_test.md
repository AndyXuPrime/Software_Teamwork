---
sidebar_position: 10
slug: /run_retrieval_test
sidebar_custom_props: {
  categoryIcon: LucideTextSearch
}
---
# Run retrieval test

Conduct a retrieval test on your dataset to check whether the intended chunks can be retrieved.

---

After your files are uploaded and parsed, run a retrieval test to validate chunking, embedding, and hybrid search settings before wiring the runtime into downstream applications. Dataset parser settings, similarity weights, rerank model choice, and LLM configuration all affect retrieval quality. A retrieval test verifies whether the intended chunks can be recovered so you can isolate parser/RAG issues from upstream LLM limitations.

During a retrieval test, chunks created from your specified chunking method are retrieved using a hybrid search. This search combines weighted keyword similarity with either weighted vector cosine similarity or a weighted reranking score, depending on your settings:

- If no rerank model is selected, weighted keyword similarity will be combined with weighted vector cosine similarity.
- If a rerank model is selected, weighted keyword similarity will be combined with weighted vector reranking score.

In contrast, chunks created from [knowledge graph construction](./advanced/construct_knowledge_graph.md) are retrieved solely using vector cosine similarity.

## Prerequisites

- Your files are uploaded and successfully parsed before running a retrieval test.
- A knowledge graph must be successfully built before enabling **Use knowledge graph**.

## Configurations

### Similarity threshold

This sets the bar for retrieving chunks: chunks with similarities below the threshold will be filtered out. By default, the threshold is set to 0.2. This means that only chunks with hybrid similarity score of 20 or higher will be retrieved.

### Vector similarity weight

This sets the weight of vector similarity in the composite similarity score, whether used with vector cosine similarity or a reranking score. By default, it is set to 0.3, making the weight of the other component 0.7 (1 - 0.3).

### Rerank model

- If left empty, RAGFlow will use a combination of weighted keyword similarity and weighted vector cosine similarity.
- If a rerank model is selected, weighted keyword similarity will be combined with weighted vector reranking score.

:::danger IMPORTANT
Using a rerank model will significantly increase the time to receive a response.
:::

### Use knowledge graph

In a knowledge graph, an entity description, a relationship description, or a community report each exists as an independent chunk. This switch indicates whether to add these chunks to the retrieval.

The switch is disabled by default. When enabled, RAGFlow performs graph-aware retrieval over entity and relationship chunks before returning results.

:::danger IMPORTANT
Using a knowledge graph in a retrieval test will significantly increase the time to receive a response.
:::

### Cross-language search

To perform a [cross-language search](../../references/glossary.mdx#cross-language-search), select one or more target languages. The configured default chat model translates the query into the selected target language(s) to improve cross-language semantic matching.

:::tip NOTE
- When selecting target languages, ensure those languages are present in the dataset.
- If no target language is selected, the system searches using the query language only.
:::

### Test text

This field is where you put in your testing query.

## Procedure

1. Open the dataset retrieval test surface, enter your query in **Test text**, and run the test.
2. If the results are unsatisfactory, tune the options listed in the Configuration section and rerun the test.

:::caution WARNING
Adjusted similarity weights, thresholds, and rerank settings must be persisted explicitly in dataset or runtime configuration; they are not saved automatically by the test UI alone.
:::

## Frequently asked questions

### Is an LLM used when the Use Knowledge Graph switch is enabled?

Yes. The configured LLM analyzes the query and extracts related entities and relationships from the knowledge graph, which increases token usage and latency.
