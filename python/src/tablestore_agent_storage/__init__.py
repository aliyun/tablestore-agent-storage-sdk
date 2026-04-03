"""Tablestore Agent Storage SDK for Python."""

from tablestore_agent_storage.client import AgentStorageClient
from tablestore_agent_storage.models import (
    # Enums
    MetadataFieldType,
    SearchType,
    RerankingType,
    RetrievalQueryType,
    MetadataFilter,
    # Knowledge Base Models
    MetadataField,
    EmbeddingConfiguration,
    CreateKnowledgeBaseRequest,
    DeleteKnowledgeBaseRequest,
    DescribeKnowledgeBaseRequest,
    ListKnowledgeBaseRequest,
    UpdateKnowledgeBaseRequest,
    # Document Models
    AddDocumentItem,
    UploadDocumentItem,
    DeleteDocumentItem,
    AddDocumentsRequest,
    UploadDocumentsRequest,
    UpdateDocumentRequest,
    GetDocumentRequest,
    ListDocumentsRequest,
    DeleteDocumentsRequest,
    # Chunk Models
    UpdateChunkItem,
    UpdateChunksRequest,
    ListChunksRequest,
    # Retrieve Models
    RetrievalQuery,
    DenseVectorSearchConfiguration,
    FulltextSearchConfiguration,
    RRFConfiguration,
    ModelConfiguration,
    WeightConfiguration,
    RerankingConfiguration,
    RetrievalConfiguration,
    RetrieveRequest,
)

__version__ = "1.0.4"
__all__ = [
    # Client
    "AgentStorageClient",
    # Enums
    "MetadataFieldType",
    "SearchType",
    "RerankingType",
    "RetrievalQueryType",
    # Filter Models
    "MetadataFilter",
    # Knowledge Base Models
    "MetadataField",
    "EmbeddingConfiguration",
    "CreateKnowledgeBaseRequest",
    "DeleteKnowledgeBaseRequest",
    "DescribeKnowledgeBaseRequest",
    "ListKnowledgeBaseRequest",
    "UpdateKnowledgeBaseRequest",
    # Document Models
    "AddDocumentItem",
    "UploadDocumentItem",
    "DeleteDocumentItem",
    "AddDocumentsRequest",
    "UploadDocumentsRequest",
    "UpdateDocumentRequest",
    "GetDocumentRequest",
    "ListDocumentsRequest",
    "DeleteDocumentsRequest",
    # Chunk Models
    "UpdateChunkItem",
    "UpdateChunksRequest",
    "ListChunksRequest",
    # Retrieve Models
    "RetrievalQuery",
    "DenseVectorSearchConfiguration",
    "FulltextSearchConfiguration",
    "RRFConfiguration",
    "ModelConfiguration",
    "WeightConfiguration",
    "RerankingConfiguration",
    "RetrievalConfiguration",
    "RetrieveRequest",
]
