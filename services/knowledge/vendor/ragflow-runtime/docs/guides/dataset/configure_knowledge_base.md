---
sidebar_position: -10
slug: /configure_knowledge_base
sidebar_custom_props: {
  categoryIcon: LucideCog
}
---
# Configure dataset

Each RAGFlow dataset serves as a knowledge source, parsing files into chunks and indexes that can be retrieved later. This guide demonstrates some basic usages of the dataset feature, covering the following topics:

- Create a dataset
- Configure a dataset
- Search for a dataset
- Delete a dataset

## Create dataset

With multiple datasets, you can build more flexible, diversified question answering. To create your first dataset:

![create dataset](https://raw.githubusercontent.com/infiniflow/ragflow-docs/main/images/create_knowledge_base.jpg)

_Each time a dataset is created, a folder with the same name is generated in the **root/.knowledgebase** directory._

## Configure dataset

The following screenshot shows the configuration page of a dataset. A proper configuration of your dataset is crucial for future AI chats. For example, choosing the wrong embedding model or chunking method would cause unexpected semantic loss or mismatched answers in chats.

![dataset configuration](https://raw.githubusercontent.com/infiniflow/ragflow-docs/main/images/configure_knowledge_base.jpg)

This section covers the following topics:

- Select chunking method
- Select embedding model
- Upload file
- Parse file
- Intervene with file parsing results
- Run retrieval testing

### Select chunking method

RAGFlow offers multiple built-in chunking template to facilitate chunking files of different layouts and ensure semantic integrity. From the **Built-in** chunking method dropdown under **Parse type**, you can choose the default template that suits the layouts and formats of your files. The following table shows the descriptions and the compatible file formats of each supported chunk template:

| **Template** | Description                                                                   | File format                                                                                             |
|--------------|-------------------------------------------------------------------------------|---------------------------------------------------------------------------------------------------------|
| General      | Files are consecutively chunked based on a preset chunk token number.         | MD, MDX, DOCX, XLSX, XLS (Excel 97-2003), PPT, PDF, TXT, JPEG, JPG, PNG, TIF, GIF, CSV, JSON, EML, HTML |
| Q&A          | Retrieves relevant information and generates answers to respond to questions. | XLSX, XLS (Excel 97-2003), CSV/TXT                                                                      |
| Resume       | Enterprise edition only. You can also try it out on cloud.ragflow.io.          | DOCX, PDF, TXT                                                                                          |
| Manual       |                                                                               | PDF                                                                                                     |
| Table        | The table mode uses TSI technology for efficient data parsing.                | XLSX, XLS (Excel 97-2003), CSV/TXT                                                                      |
| Paper        |                                                                               | PDF                                                                                                     |
| Book         |                                                                               | DOCX, PDF, TXT                                                                                          |
| Laws         |                                                                               | DOCX, PDF, TXT                                                                                          |
| Presentation |                                                                               | PDF, PPTX                                                                                               |
| Picture      |                                                                               | JPEG, JPG, PNG, TIF, GIF                                                                                |
| One          | Each document is chunked in its entirety (as one).                            | DOCX, XLSX, XLS (Excel 97-2003), PDF, TXT                                                               |
| Tag          | The dataset functions as a tag set for the others.                            | XLSX, CSV/TXT                                                                                           |

You can also change a file's chunking method on the **Files** page.

![change chunking method](https://raw.githubusercontent.com/infiniflow/ragflow-docs/main/images/change_chunking_method.jpg)

### Select embedding model

An embedding model converts chunks into embeddings. It cannot be changed once the dataset has chunks. To switch to a different embedding model, you must delete all existing chunks in the dataset. The obvious reason is that we *must* ensure that files in a specific dataset are converted to embeddings using the *same* embedding model (ensure that they are compared in the same embedding space).

:::danger IMPORTANT
Some embedding models are optimized for specific languages, so performance may be compromised if you use them to embed documents in other languages.
:::

### Upload file

- In **Knowledge Base**, you can upload a single file or a folder of files to a dataset for parsing.

### Parse file

File parsing is a crucial topic in dataset configuration. The meaning of file parsing in RAGFlow is twofold: chunking files based on file layout and building embedding and full-text (keyword) indexes on these chunks. After having selected the chunking method and embedding model, you can start parsing a file:

![parse file](https://raw.githubusercontent.com/infiniflow/ragflow-docs/main/images/parse_file.jpg)

- As shown above, RAGFlow allows you to use a different chunking method for a particular file, offering flexibility beyond the default method.
- As shown above, RAGFlow allows you to enable or disable individual files, offering finer control over dataset-based AI chats.

### Intervene with file parsing results

RAGFlow features visibility and explainability, allowing you to view the chunking results and intervene where necessary. To do so:

1. Click on the file that completes file parsing to view the chunking results:

   _You are taken to the **Chunk** page:_

   ![chunks](https://raw.githubusercontent.com/infiniflow/ragflow-docs/main/images/file_chunks.jpg)

2. Hover over each snapshot for a quick view of each chunk.

3. Double-click the chunked texts to add keywords, questions, tags, or make *manual* changes where necessary:

   ![update chunk](https://raw.githubusercontent.com/infiniflow/ragflow-docs/main/images/add_keyword_question.jpg)

:::caution NOTE
You can add keywords to a file chunk to increase its ranking for queries containing those keywords. This action increases its keyword weight and can improve its position in search list.
:::

4. In Retrieval testing, ask a quick question in **Test text** to double-check if your configurations work:

   _As you can tell from the following, RAGFlow responds with truthful citations._

   ![retrieval test](https://raw.githubusercontent.com/infiniflow/ragflow-docs/main/images/retrieval_test.jpg)

### Run retrieval testing

RAGFlow uses multiple recall of both full-text search and vector search in its chats. Prior to setting up an AI chat, consider adjusting the following parameters to ensure that the intended information always turns up in answers:

- Similarity threshold: Chunks with similarities below the threshold will be filtered. By default, it is set to 0.2.
- Vector similarity weight: The percentage by which vector similarity contributes to the overall score. By default, it is set to 0.3.

See [Run retrieval test](./run_retrieval_test.md) for details.

## Search for dataset

As of RAGFlow v0.26.2, the search feature is still in a rudimentary form, supporting only dataset search by name.

![search dataset](https://raw.githubusercontent.com/infiniflow/ragflow-docs/main/images/search_datasets.jpg)

## Delete dataset

You are allowed to delete a dataset. Hover your mouse over the three dot of the intended dataset card and the **Delete** option appears. Once you delete a dataset, the associated folder under **root/.knowledge** directory is AUTOMATICALLY REMOVED. The consequence is:

- The files uploaded directly to the dataset are gone;
- The file references, which you created from within RAGFlow's File system, are gone, but the associated files still exist.

![delete dataset](https://raw.githubusercontent.com/infiniflow/ragflow-docs/main/images/delete_datasets.jpg)
