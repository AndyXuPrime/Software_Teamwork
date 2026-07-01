//
//  Copyright 2026 The InfiniFlow Authors. All Rights Reserved.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.
//

package router

import (
	"github.com/gin-gonic/gin"

	"ragflow/internal/common"
	"ragflow/internal/handler"
)

type Router struct {
	authHandler          *handler.AuthHandler
	tenantHandler        *handler.TenantHandler
	documentHandler      *handler.DocumentHandler
	datasetsHandler      *handler.DatasetsHandler
	systemHandler        *handler.SystemHandler
	knowledgebaseHandler *handler.KnowledgebaseHandler
	chunkHandler         *handler.ChunkHandler
	fileHandler          *handler.FileHandler
	mcpHandler           *handler.MCPHandler
	providerHandler      *handler.ProviderHandler
	modelHandler         *handler.ModelHandler
}

// NewRouter create router
func NewRouter(
	authHandler *handler.AuthHandler,
	tenantHandler *handler.TenantHandler,
	documentHandler *handler.DocumentHandler,
	datasetsHandler *handler.DatasetsHandler,
	systemHandler *handler.SystemHandler,
	knowledgebaseHandler *handler.KnowledgebaseHandler,
	chunkHandler *handler.ChunkHandler,
	fileHandler *handler.FileHandler,
	mcpHandler *handler.MCPHandler,
	providerHandler *handler.ProviderHandler,
	modelHandler *handler.ModelHandler,
) *Router {
	return &Router{
		authHandler:          authHandler,
		tenantHandler:        tenantHandler,
		documentHandler:      documentHandler,
		datasetsHandler:      datasetsHandler,
		systemHandler:        systemHandler,
		knowledgebaseHandler: knowledgebaseHandler,
		chunkHandler:         chunkHandler,
		fileHandler:          fileHandler,
		mcpHandler:           mcpHandler,
		providerHandler:      providerHandler,
		modelHandler:         modelHandler,
	}
}

// Setup setup routes
func (r *Router) Setup(engine *gin.Engine) {
	// Mark all responses from Go with a header for debugging.
	engine.Use(func(c *gin.Context) {
		c.Header("X-API-Source", "go")
		c.Next()
	})

	// Log all HTTP requests.
	engine.Use(common.GinLogger())

	// Health check
	engine.GET("/health", r.systemHandler.Health)

	// System endpoints
	engine.GET("/v1/system/configs", r.systemHandler.GetConfigs)

	apiNoAuth := engine.Group("/api/v1")
	{
		apiNoAuth.GET("/system/ping", r.systemHandler.Ping)
		apiNoAuth.GET("/system/version", r.systemHandler.GetVersion)
		apiNoAuth.GET("/system/healthz", r.systemHandler.Healthz)
	}

	// Beta-token routes retained for document preview/image endpoints.
	apiBetaAuth := engine.Group("/api/v1")
	apiBetaAuth.Use(r.authHandler.BetaAuthMiddleware())
	{
		apiBetaAuth.GET("/documents/images/:image_id", r.documentHandler.GetDocumentImage)
		apiBetaAuth.GET("/documents/:id/preview", r.documentHandler.GetDocumentPreview)
		apiBetaAuth.GET("/thumbnails", r.documentHandler.GetThumbnail)
	}

	// Protected routes
	authorized := engine.Group("")
	authorized.Use(r.authHandler.AuthMiddleware())
	{
		// API v1 route group
		v1 := authorized.Group("/api/v1")
		{
			// Document routes
			documents := v1.Group("/documents")
			{
				documents.POST("", r.documentHandler.CreateDocument)
				documents.POST("/upload", r.documentHandler.UploadInfo)
				documents.GET("", r.documentHandler.ListDocuments)
				documents.GET("/:id", r.documentHandler.GetDocumentByID)
				documents.PUT("/:id", r.documentHandler.UpdateDocument)
				documents.DELETE("/:id", r.documentHandler.DeleteDocument)
				documents.POST("/ingest", r.documentHandler.Ingest)
			}

			// Dataset routes
			datasets := v1.Group("/datasets")
			{
				datasets.GET("", r.datasetsHandler.ListDatasets)
				datasets.GET("/tags/aggregation", r.datasetsHandler.AggregateTags)
				datasets.GET("/:dataset_id", r.datasetsHandler.GetDataset)
				datasets.PUT("/:dataset_id", r.datasetsHandler.UpdateDataset)
				datasets.GET("/:dataset_id/graph", r.datasetsHandler.GetKnowledgeGraph)
				datasets.GET("/:dataset_id/tags", r.datasetsHandler.ListTags)
				datasets.PUT("/:dataset_id/tags", r.datasetsHandler.RenameTag)
				datasets.DELETE("/:dataset_id/tags", r.datasetsHandler.RemoveTags)
				datasets.POST("/:dataset_id/embedding", r.datasetsHandler.RunEmbedding)
				datasets.POST("/:dataset_id/embedding/check", r.datasetsHandler.CheckEmbedding)
				datasets.POST("/:dataset_id/documents/batch-update-status", r.documentHandler.BatchUpdateDocumentStatus)
				datasets.GET("/:dataset_id/index", r.datasetsHandler.TraceIndex)
				datasets.POST("/:dataset_id/index", r.datasetsHandler.RunIndex)
				datasets.DELETE("/:dataset_id/index", r.datasetsHandler.DeleteIndex)
				datasets.DELETE("/:dataset_id/:index_type", r.datasetsHandler.DeleteIndex)
				//datasets.DELETE("/:dataset_id/graph", r.datasetsHandler.DeleteKnowledgeGraph)
				datasets.POST("", r.datasetsHandler.CreateDataset)
				datasets.DELETE("", r.datasetsHandler.DeleteDatasets)
				datasets.POST("/search", r.datasetsHandler.SearchDatasets)
				datasets.POST("/:dataset_id/search", r.datasetsHandler.SearchDataset)
				datasets.GET("/metadata/flattened", r.datasetsHandler.ListMetadataFlattened)
				datasets.GET("/:dataset_id/metadata/summary", r.documentHandler.MetadataSummaryByDataset)

				// Dataset ingestion logs
				datasets.GET("/:dataset_id/ingestions/summary", r.datasetsHandler.GetIngestionSummary)
				datasets.GET("/:dataset_id/ingestions", r.datasetsHandler.ListIngestionLogs)
				datasets.GET("/:dataset_id/ingestions/:log_id", r.datasetsHandler.GetIngestionLog)

				// Metadata Config
				datasets.GET("/:dataset_id/metadata/config", r.datasetsHandler.GetMetadataConfig)
				datasets.PUT("/:dataset_id/metadata/config", r.datasetsHandler.UpdateMetadataConfig)

				// Dataset documents
				datasets.GET("/:dataset_id/documents", r.documentHandler.ListDocuments)
				datasets.POST("/:dataset_id/documents", r.documentHandler.UploadDocuments)
				datasets.GET("/:dataset_id/documents/:document_id", r.documentHandler.DownloadDocument)
				datasets.PATCH("/:dataset_id/documents/:document_id", r.documentHandler.UpdateDatasetDocument)
				datasets.DELETE("/:dataset_id/documents", r.documentHandler.DeleteDocuments)
				datasets.POST("/:dataset_id/documents/:document_id/chunks", r.chunkHandler.AddChunk)

				// Dataset document chunk
				datasets.GET("/:dataset_id/documents/:document_id/chunks", r.chunkHandler.ListChunks)
				datasets.PATCH("/:dataset_id/documents/:document_id/chunks", r.chunkHandler.SwitchChunks)
				datasets.GET("/:dataset_id/documents/:document_id/chunks/:chunk_id", r.chunkHandler.Get)
				datasets.POST("/:dataset_id/chunks", r.chunkHandler.Parse)
				datasets.PATCH("/:dataset_id/documents/:document_id/chunks/:chunk_id", r.chunkHandler.UpdateChunk)
				datasets.POST("/:dataset_id/documents/parse", r.documentHandler.StartIngestionTask)
				datasets.GET("/ingestion/tasks", r.documentHandler.ListIngestionTasks)
				datasets.PUT("/ingestion/tasks", r.documentHandler.StopIngestionTasks)
				datasets.DELETE("/ingestion/tasks", r.documentHandler.RemoveIngestionTasks)
				//datasets.POST("/:dataset_id/documents/parse", r.documentHandler.ParseDocuments)
				//datasets.POST("/:dataset_id/documents/stop", r.documentHandler.StopParseDocuments)
				datasets.DELETE("/:dataset_id/chunks", r.chunkHandler.StopParsing)
				datasets.DELETE("/:dataset_id/documents/:document_id/chunks", r.chunkHandler.RemoveChunks)
				datasets.PUT("/:dataset_id/documents/:document_id/metadata/config", r.datasetsHandler.UpdateDocumentMetadataConfig)
				datasets.POST("/:dataset_id/metadata/update", r.documentHandler.MetadataBatchUpdate)
				datasets.PATCH("/:dataset_id/documents/metadatas", r.documentHandler.UpdateDocumentMetadatas)
			}

			file := v1.Group("/files")
			{
				file.POST("", r.fileHandler.UploadFile)
				file.GET("", r.fileHandler.ListFiles)
				file.DELETE("", r.fileHandler.DeleteFiles)
				file.POST("/move", r.fileHandler.MoveFiles)
				file.POST("/link-to-datasets", r.fileHandler.LinkToDatasets)
				file.GET("/:id/ancestors", r.fileHandler.GetFileAncestors)
				file.GET("/:id/parent", r.fileHandler.GetParentFolder)
				file.GET("/:id", r.fileHandler.Download)
			}

			// Author routes
			authors := v1.Group("/authors")
			{
				authors.GET("/:author_id/documents", r.documentHandler.GetDocumentsByAuthorID)
			}

			// provider pool route group
			provider := v1.Group("/providers")
			{
				provider.GET("/", r.providerHandler.ListProviders)
				provider.PUT("/", r.providerHandler.AddProvider)
				provider.GET("/:provider_name", r.providerHandler.ShowProvider)
				provider.DELETE("/:provider_name", r.providerHandler.DeleteProvider)
				provider.GET("/:provider_name/models", r.providerHandler.ListModels)
				provider.GET("/:provider_name/models/:model_name", r.providerHandler.ShowModel)
				provider.POST("/:provider_name/instances", r.providerHandler.CreateProviderInstance)
				provider.GET("/:provider_name/instances", r.providerHandler.ListProviderInstances)
				provider.GET("/:provider_name/instances/:instance_name", r.providerHandler.ShowProviderInstance)
				provider.GET("/:provider_name/instances/:instance_name/balance", r.providerHandler.ShowInstanceBalance)
				provider.GET("/:provider_name/instances/:instance_name/connection", r.providerHandler.CheckInstanceConnection)
				provider.POST("/:provider_name/connection", r.providerHandler.CheckConnection)
				provider.GET("/:provider_name/instances/:instance_name/tasks", r.providerHandler.ListTasks)
				provider.GET("/:provider_name/instances/:instance_name/tasks/:task_id", r.providerHandler.ShowTask)
				provider.PUT("/:provider_name/instances/:instance_name", r.providerHandler.AlterProviderInstance)
				provider.DELETE("/:provider_name/instances", r.providerHandler.DropProviderInstance)
				provider.GET("/:provider_name/instances/:instance_name/models", r.providerHandler.ListInstanceModels)
				provider.PATCH("/:provider_name/instances/:instance_name/models/*model_name", r.providerHandler.EnableOrDisableModel)
				provider.POST("/:provider_name/instances/:instance_name/models", r.providerHandler.AddModel)
				provider.DELETE("/:provider_name/instances/:instance_name/models", r.providerHandler.DropInstanceModels)
				v1.POST("/chat/completions", r.providerHandler.ChatToModel)
				v1.POST("/embeddings", r.providerHandler.EmbedText)
				v1.POST("/rerank", r.providerHandler.RerankDocument)
				v1.POST("/audio/transcriptions", r.providerHandler.TranscribeAudio)
				v1.POST("/audio/speech", r.providerHandler.AudioSpeech)
				v1.POST("/file/ocr", r.providerHandler.OCRFile)
				v1.POST("/file/parse", r.providerHandler.ParseFile)
			}

			model := v1.Group("/models")
			{
				// GET /models returns the tenant's added models across
				// all instances, matching Python's
				// models_api_service.list_tenant_added_models. Front-end
				// useFetchAllAddedModels consumes this. Routed to the
				// provider handler because that's where the
				// modelProviderService is wired.
				model.GET("/", r.providerHandler.ListTenantAddedModels)

				// TODO: list default models?
				//model.GET("/", r.tenantHandler.GetModels)
				model.PATCH("/", r.tenantHandler.SetModels)
				// Tenant default-model selection. Mirrors the Python contract at
				// api/apps/restful_apis/models_api.py:84.
				model.GET("/default", r.tenantHandler.GetDefaultModels)
				model.PATCH("/default", r.tenantHandler.SetDefaultModels)
			}

			allModels := v1.Group("/all-models")
			{
				allModels.GET("", r.modelHandler.ListAllModels)
				allModels.GET("/:model_name", r.modelHandler.ShowModel)
			}

			// MCP server routes. Per-server CRUD ships via separate PRs that
			// share the same handler/service: GET list (#15253), GET by id
			// (#15254), POST create (#15260, merged), PUT (#15261), DELETE
			// (#15262, merged). This PR adds only the non-overlapping
			// endpoints: import and test.
			mcp := v1.Group("/mcp")
			{
				mcp.POST("/servers", r.mcpHandler.CreateMCPServer)
				mcp.GET("/servers", r.mcpHandler.ListMCPServers)
				mcp.GET("/servers/:mcp_id", r.mcpHandler.GetMCPServer)
				mcp.PUT("/servers/:mcp_id", r.mcpHandler.UpdateMCPServer)
				mcp.DELETE("/servers/:mcp_id", r.mcpHandler.DeleteMCPServer)
				mcp.POST("/servers/import", r.mcpHandler.ImportMCPServers)
				mcp.POST("/servers/:mcp_id/test", r.mcpHandler.TestMCPServer)
			}

			system := v1.Group("/system")
			{
				system.GET("/configs", r.systemHandler.GetConfigs)
				system.GET("/status", r.systemHandler.GetStatus)
				system.GET("/stats", r.systemHandler.GetStats)

				config := system.Group("/config")
				{
					config.GET("/log", r.systemHandler.GetLogLevel)
					config.PUT("/log", r.systemHandler.SetLogLevel)
				}

				// Variables/Settings
				system.GET("/variables", r.systemHandler.ListVariables)
				system.PUT("/variables", r.systemHandler.SetVariable)
				system.GET("/variables/:var_name", r.systemHandler.ShowVariable)

				// Environments
				system.GET("/environments", r.systemHandler.ListEnvironments)

				//log := system.Group("/log")
				//{
				//	// /api/v1/system/log GET
				//	log.GET("", r.systemHandler.GetLogLevel)
				//	// /api/v1/system/log PUT
				//	log.PUT("", r.systemHandler.SetLogLevel)
				//}

				tokens := system.Group("/tokens")
				{
					// list tokens /api/v1/system/tokens GET
					tokens.GET("", r.systemHandler.ListAPIKeys)
					// create token /api/v1/system/tokens POST
					tokens.POST("", r.systemHandler.CreateKey)
					// delete token /api/v1/system/tokens/:key DELETE
					tokens.DELETE("/:key", r.systemHandler.DeleteKey)
				}

				keys := system.Group("/keys")
				{
					// list keys /api/v1/system/keys GET
					keys.GET("", r.systemHandler.ListAPIKeys)
					// create key /api/v1/system/keys POST
					keys.POST("", r.systemHandler.CreateKey)
					// delete key /api/v1/system/keys/:key DELETE
					keys.DELETE("/:key", r.systemHandler.DeleteKey)
				}
			}
		}

		// Knowledge base routes
		kb := v1.Group("/kb")
		{
			kb.POST("/update", r.knowledgebaseHandler.UpdateKB)
			kb.POST("/update_metadata_setting", r.knowledgebaseHandler.UpdateMetadataSetting)
			kb.GET("/detail", r.knowledgebaseHandler.GetDetail)
			kb.GET("/tags", r.knowledgebaseHandler.ListTagsFromKbs)
			kb.GET("/get_meta", r.knowledgebaseHandler.GetMeta)
			kb.GET("/basic_info", r.knowledgebaseHandler.GetBasicInfo)

			// KB ID specific routes
			kbByID := kb.Group("/:kb_id")
			{
				kbByID.GET("/tags", r.knowledgebaseHandler.ListTags)
				kbByID.POST("/rename_tag", r.knowledgebaseHandler.RenameTag)
				kbByID.GET("/knowledge_graph", r.knowledgebaseHandler.KnowledgeGraph)
				kbByID.DELETE("/knowledge_graph", r.knowledgebaseHandler.DeleteKnowledgeGraph)
			}
		}

		// Tenant routes (per-tenant resources)
		tenant := v1.Group("/tenant")
		{
			tenant.POST("/chunk_store", r.tenantHandler.CreateChunkStore)                     // Internal API only for GO
			tenant.DELETE("/chunk_store", r.tenantHandler.DeleteChunkStore)                   // Internal API only for GO
			tenant.POST("/metadata_store", r.tenantHandler.CreateMetadataStore)               // Internal API only for GO
			tenant.DELETE("/metadata_store", r.tenantHandler.DeleteMetadataStore)             // Internal API only for GO
			tenant.POST("/insert_chunks_from_file", r.tenantHandler.InsertChunksFromFile)     // Internal API only for GO
			tenant.POST("/insert_metadata_from_file", r.tenantHandler.InsertMetadataFromFile) // Internal API only for GO
		}

		// Document routes
		doc := v1.Group("/document")
		{
			doc.POST("/list", r.documentHandler.ListDocuments)
			doc.POST("/metadata/summary", r.documentHandler.MetadataSummary)
			doc.POST("/set_meta", r.documentHandler.SetMeta)
			doc.POST("/delete_meta", r.documentHandler.DeleteMeta) // Internal API only for GO
		}

		// Chunk routes
		chunk := v1.Group("/chunk")
		{
			chunk.POST("/list", r.chunkHandler.List)
			chunk.POST("/update", r.chunkHandler.UpdateChunk) // Internal API only for GO
		}

		// File routes
		file := authorized.Group("/v1/file")
		{
			file.GET("/root_folder", r.fileHandler.GetRootFolder)
			file.GET("/parent_folder", r.fileHandler.GetParentFolder)
			file.GET("/all_parent_folder", r.fileHandler.GetAllParentFolders)
		}

	}

	// Handle undefined routes
	engine.NoRoute(handler.HandleNoRoute)
}
