"""Model classes for Agent Storage SDK requests."""

from dataclasses import dataclass, field, asdict
from enum import Enum
from typing import Any, Dict, List, Optional, TypedDict, Union


# ============================================================================
# Sentinel Value
# ============================================================================

class _Unset:
    """Sentinel class to distinguish between 'not set' and 'set to None'."""
    _instance = None

    def __new__(cls):
        if cls._instance is None:
            cls._instance = super().__new__(cls)
        return cls._instance

    def __repr__(self):
        return "UNSET"

    def __bool__(self):
        return False

UNSET = _Unset()
"""Sentinel value indicating a field was not explicitly set by the user."""


# ============================================================================
# Item Request Models
# ============================================================================

class ItemScope(TypedDict):
    """Exact or wildcard four-part scope used by the item APIs."""

    appId: str
    tenantId: str
    agentId: str
    runId: str


class ItemPrecondition(TypedDict):
    """Complete resource-version precondition for exact Update/Delete."""

    expectedVersionId: str


class _ItemScopedRequestRequired(TypedDict):
    memoryStoreName: str
    scope: ItemScope


class ItemScopedRequest(_ItemScopedRequestRequired, total=False):
    """Fields shared by item requests; the SDK fills the optional type."""

    type: str


class GetItemRequest(ItemScopedRequest, total=False):
    """Get request; exactly one of itemId and path is required by the service."""

    itemId: str
    path: str
    view: str


class _UpdateItemRequestRequired(ItemScopedRequest):
    itemId: str


class UpdateItemRequest(_UpdateItemRequestRequired, total=False):
    """Update request; content, target path, or both are required by the service."""

    content: str
    path: str
    precondition: ItemPrecondition
    overwrite: bool
    sessionId: str
    view: str


class DeleteItemRequest(ItemScopedRequest, total=False):
    """Exact single-item or wildcard ranged-delete request."""

    itemId: str
    path: str
    pathPrefix: str
    precondition: ItemPrecondition
    sessionId: str


# ============================================================================
# Enums
# ============================================================================

class MetadataFieldType(str, Enum):
    """Supported metadata field types."""
    STRING = "string"
    LONG = "long"
    DOUBLE = "double"
    BOOLEAN = "boolean"
    DATE = "date"
    STRING_LIST = "string_list"
    LONG_LIST = "long_list"
    DOUBLE_LIST = "double_list"
    BOOLEAN_LIST = "boolean_list"
    DATE_LIST = "date_list"

class SearchType(str, Enum):
    """Supported search types for retrieval."""
    DENSE_VECTOR = "DENSE_VECTOR"
    FULL_TEXT = "FULL_TEXT"

class RerankingType(str, Enum):
    """Supported reranking types."""
    RRF = "RRF"
    MODEL = "MODEL"
    WEIGHT = "WEIGHT"

class RetrievalQueryType(str, Enum):
    """Supported retrieval query types."""
    TEXT = "TEXT"

# ============================================================================
# Base Classes
# ============================================================================

@dataclass
class BaseModel:
    """Base class for all models with common serialization methods."""

    def to_dict(self) -> Dict[str, Any]:
        """Convert model to dictionary, excluding None values."""
        result = {}
        for key, value in asdict(self).items():
            if value is not None:
                if isinstance(value, Enum):
                    result[key] = value.value
                elif isinstance(value, list):
                    result[key] = [
                        item.value if isinstance(item, Enum)
                        else (item.to_dict() if hasattr(item, 'to_dict') else item)
                        for item in value
                    ]
                elif hasattr(value, 'to_dict'):
                    result[key] = value.to_dict()
                else:
                    result[key] = value
        return result

# ============================================================================
# Metadata Filter Models
# ============================================================================

@dataclass
class KeyValueCondition:
    """Condition with key and value."""
    key: str
    value: Any


class MetadataFilter:
    """
    Metadata filter for knowledge base retrieval.
    Supports various filter operators for metadata-based filtering.
    A MetadataFilter can only contain ONE type of operator at a time.
    
    Usage examples:
        # Basic comparison
        filter1 = MetadataFilter.equals("name", "test")
        filter2 = MetadataFilter.greater_than("age", 18)
        
        # Logical operators
        filter3 = MetadataFilter.and_all(
            MetadataFilter.equals("name", "test"),
            MetadataFilter.greater_than("age", 18)
        )
        
        # Using builder
        filter4 = MetadataFilter.builder() \\
            .equals("name", "test") \\
            .greater_than("age", 18) \\
            .build_and()
    """
    
    def __init__(self):
        # Basic comparison operators
        self._equals: Optional[KeyValueCondition] = None
        self._not_equals: Optional[KeyValueCondition] = None
        self._greater_than: Optional[KeyValueCondition] = None
        self._greater_than_or_equals: Optional[KeyValueCondition] = None
        self._less_than: Optional[KeyValueCondition] = None
        self._less_than_or_equals: Optional[KeyValueCondition] = None
        
        # List operators
        self._in: Optional[KeyValueCondition] = None
        self._not_in: Optional[KeyValueCondition] = None
        
        # String operators
        self._starts_with: Optional[KeyValueCondition] = None
        self._string_contains: Optional[KeyValueCondition] = None
        self._list_contains: Optional[KeyValueCondition] = None
        
        # Logical operators
        self._and_all: Optional[List['MetadataFilter']] = None
        self._or_all: Optional[List['MetadataFilter']] = None
        self._not_all: Optional[List['MetadataFilter']] = None

    # Static factory methods for basic operators
    @staticmethod
    def equals(key: str, value: Union[str, int, float, bool]) -> 'MetadataFilter':
        """Create an equals filter."""
        filter_obj = MetadataFilter()
        filter_obj._equals = KeyValueCondition(key, value)
        return filter_obj

    @staticmethod
    def not_equals(key: str, value: Union[str, int, float, bool]) -> 'MetadataFilter':
        """Create a not equals filter."""
        filter_obj = MetadataFilter()
        filter_obj._not_equals = KeyValueCondition(key, value)
        return filter_obj

    @staticmethod
    def greater_than(key: str, value: Union[int, float]) -> 'MetadataFilter':
        """Create a greater than filter."""
        filter_obj = MetadataFilter()
        filter_obj._greater_than = KeyValueCondition(key, value)
        return filter_obj

    @staticmethod
    def greater_than_or_equals(key: str, value: Union[int, float]) -> 'MetadataFilter':
        """Create a greater than or equals filter."""
        filter_obj = MetadataFilter()
        filter_obj._greater_than_or_equals = KeyValueCondition(key, value)
        return filter_obj

    @staticmethod
    def less_than(key: str, value: Union[int, float]) -> 'MetadataFilter':
        """Create a less than filter."""
        filter_obj = MetadataFilter()
        filter_obj._less_than = KeyValueCondition(key, value)
        return filter_obj

    @staticmethod
    def less_than_or_equals(key: str, value: Union[int, float]) -> 'MetadataFilter':
        """Create a less than or equals filter."""
        filter_obj = MetadataFilter()
        filter_obj._less_than_or_equals = KeyValueCondition(key, value)
        return filter_obj

    @staticmethod
    def in_list(key: str, value: List[str]) -> 'MetadataFilter':
        """Create an in filter."""
        filter_obj = MetadataFilter()
        filter_obj._in = KeyValueCondition(key, value)
        return filter_obj

    @staticmethod
    def not_in_list(key: str, value: List[str]) -> 'MetadataFilter':
        """Create a not in filter."""
        filter_obj = MetadataFilter()
        filter_obj._not_in = KeyValueCondition(key, value)
        return filter_obj

    @staticmethod
    def starts_with(key: str, value: str) -> 'MetadataFilter':
        """Create a starts with filter."""
        filter_obj = MetadataFilter()
        filter_obj._starts_with = KeyValueCondition(key, value)
        return filter_obj

    @staticmethod
    def string_contains(key: str, value: str) -> 'MetadataFilter':
        """Create a string contains filter."""
        filter_obj = MetadataFilter()
        filter_obj._string_contains = KeyValueCondition(key, value)
        return filter_obj

    @staticmethod
    def list_contains(key: str, value: str) -> 'MetadataFilter':
        """Create a list contains filter."""
        filter_obj = MetadataFilter()
        filter_obj._list_contains = KeyValueCondition(key, value)
        return filter_obj

    @staticmethod
    def and_all(*filters: 'MetadataFilter') -> 'MetadataFilter':
        """Create an AND filter combining multiple conditions."""
        filter_obj = MetadataFilter()
        filter_obj._and_all = list(filters)
        return filter_obj

    @staticmethod
    def or_all(*filters: 'MetadataFilter') -> 'MetadataFilter':
        """Create an OR filter combining multiple conditions."""
        filter_obj = MetadataFilter()
        filter_obj._or_all = list(filters)
        return filter_obj

    @staticmethod
    def not_all(*filters: 'MetadataFilter') -> 'MetadataFilter':
        """Create a NOT filter combining multiple conditions."""
        filter_obj = MetadataFilter()
        filter_obj._not_all = list(filters)
        return filter_obj

    @staticmethod
    def builder() -> 'MetadataFilterBuilder':
        """Create a builder for constructing complex filters."""
        return MetadataFilterBuilder()

    def to_dict(self) -> Dict[str, Any]:
        """Convert filter to dictionary for JSON serialization."""
        result = {}
        
        if self._equals is not None:
            result["equals"] = {"key": self._equals.key, "value": self._equals.value}
        elif self._not_equals is not None:
            result["notEquals"] = {"key": self._not_equals.key, "value": self._not_equals.value}
        elif self._greater_than is not None:
            result["greaterThan"] = {"key": self._greater_than.key, "value": self._greater_than.value}
        elif self._greater_than_or_equals is not None:
            result["greaterThanOrEquals"] = {"key": self._greater_than_or_equals.key, "value": self._greater_than_or_equals.value}
        elif self._less_than is not None:
            result["lessThan"] = {"key": self._less_than.key, "value": self._less_than.value}
        elif self._less_than_or_equals is not None:
            result["lessThanOrEquals"] = {"key": self._less_than_or_equals.key, "value": self._less_than_or_equals.value}
        elif self._in is not None:
            result["in"] = {"key": self._in.key, "value": self._in.value}
        elif self._not_in is not None:
            result["notIn"] = {"key": self._not_in.key, "value": self._not_in.value}
        elif self._starts_with is not None:
            result["startsWith"] = {"key": self._starts_with.key, "value": self._starts_with.value}
        elif self._string_contains is not None:
            result["stringContains"] = {"key": self._string_contains.key, "value": self._string_contains.value}
        elif self._list_contains is not None:
            result["listContains"] = {"key": self._list_contains.key, "value": self._list_contains.value}
        elif self._and_all is not None:
            result["andAll"] = [f.to_dict() for f in self._and_all]
        elif self._or_all is not None:
            result["orAll"] = [f.to_dict() for f in self._or_all]
        elif self._not_all is not None:
            result["notAll"] = [f.to_dict() for f in self._not_all]
        
        return result

    def to_json(self) -> str:
        """Convert filter to JSON string."""
        import json
        return json.dumps(self.to_dict(), indent=2)


class MetadataFilterBuilder:
    """Builder class for constructing complex MetadataFilter objects."""
    
    def __init__(self):
        self._filters: List[MetadataFilter] = []

    def add(self, filter_obj: MetadataFilter) -> 'MetadataFilterBuilder':
        """Add a filter to the builder."""
        self._filters.append(filter_obj)
        return self

    def equals(self, key: str, value: Union[str, int, float, bool]) -> 'MetadataFilterBuilder':
        """Add an equals filter."""
        self._filters.append(MetadataFilter.equals(key, value))
        return self

    def not_equals(self, key: str, value: Union[str, int, float, bool]) -> 'MetadataFilterBuilder':
        """Add a not equals filter."""
        self._filters.append(MetadataFilter.not_equals(key, value))
        return self

    def greater_than(self, key: str, value: Union[int, float]) -> 'MetadataFilterBuilder':
        """Add a greater than filter."""
        self._filters.append(MetadataFilter.greater_than(key, value))
        return self

    def greater_than_or_equals(self, key: str, value: Union[int, float]) -> 'MetadataFilterBuilder':
        """Add a greater than or equals filter."""
        self._filters.append(MetadataFilter.greater_than_or_equals(key, value))
        return self

    def less_than(self, key: str, value: Union[int, float]) -> 'MetadataFilterBuilder':
        """Add a less than filter."""
        self._filters.append(MetadataFilter.less_than(key, value))
        return self

    def less_than_or_equals(self, key: str, value: Union[int, float]) -> 'MetadataFilterBuilder':
        """Add a less than or equals filter."""
        self._filters.append(MetadataFilter.less_than_or_equals(key, value))
        return self

    def in_list(self, key: str, value: List[str]) -> 'MetadataFilterBuilder':
        """Add an in filter."""
        self._filters.append(MetadataFilter.in_list(key, value))
        return self

    def not_in_list(self, key: str, value: List[str]) -> 'MetadataFilterBuilder':
        """Add a not in filter."""
        self._filters.append(MetadataFilter.not_in_list(key, value))
        return self

    def starts_with(self, key: str, value: str) -> 'MetadataFilterBuilder':
        """Add a starts with filter."""
        self._filters.append(MetadataFilter.starts_with(key, value))
        return self

    def string_contains(self, key: str, value: str) -> 'MetadataFilterBuilder':
        """Add a string contains filter."""
        self._filters.append(MetadataFilter.string_contains(key, value))
        return self

    def list_contains(self, key: str, value: str) -> 'MetadataFilterBuilder':
        """Add a list contains filter."""
        self._filters.append(MetadataFilter.list_contains(key, value))
        return self

    def build_and(self) -> MetadataFilter:
        """Build an AND filter from all added conditions."""
        if not self._filters:
            raise ValueError("Cannot build andAll filter with no conditions")
        if len(self._filters) == 1:
            return self._filters[0]
        return MetadataFilter.and_all(*self._filters)

    def build_or(self) -> MetadataFilter:
        """Build an OR filter from all added conditions."""
        if not self._filters:
            raise ValueError("Cannot build orAll filter with no conditions")
        if len(self._filters) == 1:
            return self._filters[0]
        return MetadataFilter.or_all(*self._filters)

    def build_not(self) -> MetadataFilter:
        """Build a NOT filter from all added conditions."""
        if not self._filters:
            raise ValueError("Cannot build notAll filter with no conditions")
        if len(self._filters) == 1:
            return self._filters[0]
        return MetadataFilter.not_all(*self._filters)


# Type alias for filter type (for backward compatibility and type hints)
FilterType = MetadataFilter

# ============================================================================
# Knowledge Base Models
# ============================================================================

@dataclass
class MetadataField(BaseModel):
    """Metadata field definition for knowledge base."""
    name: str
    type: MetadataFieldType

    def to_dict(self) -> Dict[str, Any]:
        return {
            "name": self.name,
            "type": self.type.value if isinstance(self.type, Enum) else self.type
        }

@dataclass
class CreateKnowledgeBaseRequest(BaseModel):
    """Request model for creating a knowledge base."""
    knowledge_base_name: str
    description: Optional[str] = None
    subspace: Optional[bool] = None
    tags: Optional[List[str]] = None
    metadata: Optional[List[MetadataField]] = None
    retrieval_configuration: Optional['RetrievalConfiguration'] = None
    embedding_configuration: Optional['EmbeddingConfiguration'] = None

    def to_dict(self) -> Dict[str, Any]:
        result = {"knowledgeBaseName": self.knowledge_base_name}
        if self.description is not None:
            result["description"] = self.description
        if self.subspace is not None:
            result["subspace"] = self.subspace
        if self.tags is not None:
            result["tags"] = self.tags
        if self.metadata is not None:
            result["metadata"] = [m.to_dict() for m in self.metadata]
        if self.retrieval_configuration is not None:
            result["retrievalConfiguration"] = self.retrieval_configuration.to_dict()
        if self.embedding_configuration is not None:
            result["embeddingConfiguration"] = self.embedding_configuration.to_dict()
        return result

@dataclass
class DeleteKnowledgeBaseRequest(BaseModel):
    """Request model for deleting a knowledge base."""
    knowledge_base_name: str

    def to_dict(self) -> Dict[str, Any]:
        return {"knowledgeBaseName": self.knowledge_base_name}

@dataclass
class DescribeKnowledgeBaseRequest(BaseModel):
    """Request model for describing a knowledge base."""
    knowledge_base_name: str

    def to_dict(self) -> Dict[str, Any]:
        return {"knowledgeBaseName": self.knowledge_base_name}

@dataclass
class ListKnowledgeBaseRequest(BaseModel):
    """Request model for listing knowledge bases."""
    max_results: Optional[int] = None
    next_token: Optional[str] = None

    def to_dict(self) -> Dict[str, Any]:
        result = {}
        if self.max_results is not None:
            result["maxResults"] = self.max_results
        if self.next_token is not None:
            result["nextToken"] = self.next_token
        return result

@dataclass
class EmbeddingConfiguration(BaseModel):
    """Configuration for embedding model used by a knowledge base."""
    provider: Optional[str] = None
    url: Optional[str] = None
    api_key: Optional[str] = None
    model: Optional[str] = None
    dimension: Optional[int] = None

    def to_dict(self) -> Dict[str, Any]:
        result = {}
        if self.provider is not None:
            result["provider"] = self.provider
        if self.url is not None:
            result["url"] = self.url
        if self.api_key is not None:
            result["apiKey"] = self.api_key
        if self.model is not None:
            result["model"] = self.model
        if self.dimension is not None:
            result["dimension"] = self.dimension
        return result

@dataclass
class UpdateKnowledgeBaseRequest(BaseModel):
    """Request model for updating a knowledge base."""
    knowledge_base_name: str
    description: Union[str, None, _Unset] = field(default_factory=lambda: UNSET)
    tags: Union[List[str], None, _Unset] = field(default_factory=lambda: UNSET)
    retrieval_configuration: Optional['RetrievalConfiguration'] = None

    def to_dict(self) -> Dict[str, Any]:
        result: Dict[str, Any] = {"knowledgeBaseName": self.knowledge_base_name}
        if not isinstance(self.description, _Unset):
            result["description"] = self.description
        if not isinstance(self.tags, _Unset):
            result["tags"] = self.tags
        if self.retrieval_configuration is not None:
            result["retrievalConfiguration"] = self.retrieval_configuration.to_dict()
        return result

# ============================================================================
# Chunk Models
# ============================================================================

@dataclass
class UpdateChunkItem(BaseModel):
    """Single chunk item for update_chunks request."""
    chunk_id: int
    doc_id: Optional[str] = None
    oss_key: Optional[str] = None
    title: Union[str, None, _Unset] = field(default_factory=lambda: UNSET)
    content: Union[str, None, _Unset] = field(default_factory=lambda: UNSET)
    status: Optional[str] = None

    def to_dict(self) -> Dict[str, Any]:
        result: Dict[str, Any] = {"chunkId": self.chunk_id}
        if self.doc_id is not None:
            result["docId"] = self.doc_id
        if self.oss_key is not None:
            result["ossKey"] = self.oss_key
        if not isinstance(self.title, _Unset):
            result["title"] = self.title
        if not isinstance(self.content, _Unset):
            result["content"] = self.content
        if self.status is not None:
            result["status"] = self.status
        return result

@dataclass
class UpdateChunksRequest(BaseModel):
    """Request model for updating chunks in a knowledge base."""
    knowledge_base_name: str
    chunks: List[UpdateChunkItem]
    subspace: Optional[str] = None

    def to_dict(self) -> Dict[str, Any]:
        result = {
            "knowledgeBaseName": self.knowledge_base_name,
            "chunks": [c.to_dict() for c in self.chunks]
        }
        if self.subspace is not None:
            result["subspace"] = self.subspace
        return result

@dataclass
class ListChunksRequest(BaseModel):
    """Request model for listing chunks in a knowledge base."""
    knowledge_base_name: str
    subspace: Optional[str] = None
    doc_id: Optional[str] = None
    oss_key: Optional[str] = None
    max_results: Optional[int] = None
    next_token: Optional[str] = None

    def to_dict(self) -> Dict[str, Any]:
        result = {"knowledgeBaseName": self.knowledge_base_name}
        if self.subspace is not None:
            result["subspace"] = self.subspace
        if self.doc_id is not None:
            result["docId"] = self.doc_id
        if self.oss_key is not None:
            result["ossKey"] = self.oss_key
        if self.max_results is not None:
            result["maxResults"] = self.max_results
        if self.next_token is not None:
            result["nextToken"] = self.next_token
        return result

# ============================================================================
# Document Models
# ============================================================================

@dataclass
class DocumentMetadata:
    """Metadata for a document."""
    data: Dict[str, Any]

    def to_dict(self) -> Dict[str, Any]:
        return self.data

@dataclass
class AddDocumentItem(BaseModel):
    """Single document item for add_documents request."""
    oss_key: str
    metadata: Optional[Dict[str, Any]] = None
    inclusion_filters: Optional[List[str]] = None
    exclusion_filters: Optional[List[str]] = None

    def to_dict(self) -> Dict[str, Any]:
        result: Dict[str, Any] = {"ossKey": self.oss_key}
        if self.metadata is not None:
            result["metadata"] = self.metadata
        if self.inclusion_filters is not None:
            result["inclusionFilters"] = self.inclusion_filters
        if self.exclusion_filters is not None:
            result["exclusionFilters"] = self.exclusion_filters
        return result

@dataclass
class UploadDocumentItem(BaseModel):
    """Single document item for upload_documents request."""
    file_path: str
    metadata: Optional[Dict[str, Any]] = None

    def to_dict(self) -> Dict[str, Any]:
        result: Dict[str, Any] = {"filePath": self.file_path}
        if self.metadata is not None:
            result["metadata"] = self.metadata
        return result

@dataclass
class DeleteDocumentItem(BaseModel):
    """Single document item for delete_documents request."""
    doc_id: Optional[str] = None
    oss_key: Optional[str] = None

    def to_dict(self) -> Dict[str, Any]:
        result = {}
        if self.doc_id is not None:
            result["docId"] = self.doc_id
        if self.oss_key is not None:
            result["ossKey"] = self.oss_key
        return result

@dataclass
class AddDocumentsRequest(BaseModel):
    """Request model for adding documents to a knowledge base."""
    knowledge_base_name: str
    documents: List[AddDocumentItem]
    subspace: Optional[str] = None

    def to_dict(self) -> Dict[str, Any]:
        result = {
            "knowledgeBaseName": self.knowledge_base_name,
            "documents": [d.to_dict() for d in self.documents]
        }
        if self.subspace is not None:
            result["subspace"] = self.subspace
        return result

@dataclass
class UploadDocumentsRequest(BaseModel):
    """Request model for uploading documents to a knowledge base."""
    knowledge_base_name: str
    documents: List[UploadDocumentItem]
    subspace: Optional[str] = None

    def to_dict(self) -> Dict[str, Any]:
        result = {
            "knowledgeBaseName": self.knowledge_base_name,
            "documents": [d.to_dict() for d in self.documents]
        }
        if self.subspace is not None:
            result["subspace"] = self.subspace
        return result

@dataclass
class GetDocumentRequest(BaseModel):
    """Request model for getting a document."""
    knowledge_base_name: str
    doc_id: Optional[str] = None
    oss_key: Optional[str] = None
    subspace: Optional[str] = None

    def to_dict(self) -> Dict[str, Any]:
        result = {"knowledgeBaseName": self.knowledge_base_name}
        if self.doc_id is not None:
            result["docId"] = self.doc_id
        if self.oss_key is not None:
            result["ossKey"] = self.oss_key
        if self.subspace is not None:
            result["subspace"] = self.subspace
        return result

@dataclass
class ListDocumentsRequest(BaseModel):
    """Request model for listing documents in a knowledge base."""
    knowledge_base_name: str
    subspace: Optional[List[str]] = None
    max_results: Optional[int] = None
    next_token: Optional[str] = None

    def to_dict(self) -> Dict[str, Any]:
        result: Dict[str, Any] = {"knowledgeBaseName": self.knowledge_base_name}
        if self.subspace is not None:
            result["subspace"] = self.subspace
        if self.max_results is not None:
            result["maxResults"] = self.max_results
        if self.next_token is not None:
            result["nextToken"] = self.next_token
        return result

@dataclass
class DeleteDocumentsRequest(BaseModel):
    """Request model for deleting documents from a knowledge base."""
    knowledge_base_name: str
    documents: List[DeleteDocumentItem]
    subspace: Optional[str] = None

    def to_dict(self) -> Dict[str, Any]:
        result = {
            "knowledgeBaseName": self.knowledge_base_name,
            "documents": [d.to_dict() for d in self.documents]
        }
        if self.subspace is not None:
            result["subspace"] = self.subspace
        return result

@dataclass
class UpdateDocumentRequest(BaseModel):
    """Request model for updating a document in a knowledge base."""
    knowledge_base_name: str
    subspace: Optional[str] = None
    oss_key: Optional[str] = None
    doc_id: Optional[str] = None
    metadata: Union[Dict[str, Any], None, _Unset] = field(default_factory=lambda: UNSET)

    def to_dict(self) -> Dict[str, Any]:
        result: Dict[str, Any] = {"knowledgeBaseName": self.knowledge_base_name}
        if self.subspace is not None:
            result["subspace"] = self.subspace
        if self.oss_key is not None:
            result["ossKey"] = self.oss_key
        if self.doc_id is not None:
            result["docId"] = self.doc_id
        if not isinstance(self.metadata, _Unset):
            result["metadata"] = self.metadata
        return result

# ============================================================================
# Retrieve Models
# ============================================================================

@dataclass
class RetrievalQuery(BaseModel):
    """Query configuration for retrieval."""
    text: str
    type: RetrievalQueryType = RetrievalQueryType.TEXT

    def to_dict(self) -> Dict[str, Any]:
        return {
            "text": self.text,
            "type": self.type.value if isinstance(self.type, Enum) else self.type
        }

@dataclass
class DenseVectorSearchConfiguration(BaseModel):
    """Configuration for dense vector search."""
    number_of_results: int = 20

    def to_dict(self) -> Dict[str, Any]:
        return {"numberOfResults": self.number_of_results}

@dataclass
class FulltextSearchConfiguration(BaseModel):
    """Configuration for fulltext search."""
    number_of_results: int = 20

    def to_dict(self) -> Dict[str, Any]:
        return {"numberOfResults": self.number_of_results}

@dataclass
class RRFConfiguration(BaseModel):
    """Configuration for RRF reranking."""
    dense_vector_search_weight: float = 1.0
    full_text_search_weight: float = 1.0
    k: int = 60

    def to_dict(self) -> Dict[str, Any]:
        return {
            "denseVectorSearchWeight": self.dense_vector_search_weight,
            "fullTextSearchWeight": self.full_text_search_weight,
            "k": self.k
        }

@dataclass
class ModelConfiguration(BaseModel):
    """Configuration for model-based reranking."""
    provider: Optional[str] = None
    model: Optional[str] = None

    def to_dict(self) -> Dict[str, Any]:
        result = {}
        if self.provider is not None:
            result["provider"] = self.provider
        if self.model is not None:
            result["model"] = self.model
        return result

@dataclass
class WeightConfiguration(BaseModel):
    """Configuration for weight-based reranking."""
    dense_vector_search_weight: Optional[float] = None
    full_text_search_weight: Optional[float] = None

    def to_dict(self) -> Dict[str, Any]:
        result = {}
        if self.dense_vector_search_weight is not None:
            result["denseVectorSearchWeight"] = self.dense_vector_search_weight
        if self.full_text_search_weight is not None:
            result["fullTextSearchWeight"] = self.full_text_search_weight
        return result

@dataclass
class RerankingConfiguration(BaseModel):
    """Configuration for reranking."""
    type: RerankingType = RerankingType.RRF
    number_of_results: int = 20
    rrf_configuration: Optional[RRFConfiguration] = None
    model_configuration: Optional[ModelConfiguration] = None
    weight_configuration: Optional[WeightConfiguration] = None

    def to_dict(self) -> Dict[str, Any]:
        result: Dict[str, Any] = {
            "type": self.type.value if isinstance(self.type, Enum) else self.type,
            "numberOfResults": self.number_of_results
        }
        if self.rrf_configuration is not None:
            result["rrfConfiguration"] = self.rrf_configuration.to_dict()
        if self.model_configuration is not None:
            result["modelConfiguration"] = self.model_configuration.to_dict()
        if self.weight_configuration is not None:
            result["weightConfiguration"] = self.weight_configuration.to_dict()
        return result

@dataclass
class RetrievalConfiguration(BaseModel):
    """Configuration for retrieval."""
    search_types: Optional[List[SearchType]] = None
    dense_vector_search_configuration: Optional[DenseVectorSearchConfiguration] = None
    fulltext_search_configuration: Optional[FulltextSearchConfiguration] = None
    reranking_configuration: Optional[RerankingConfiguration] = None
    filter: Optional[FilterType] = None

    def to_dict(self) -> Dict[str, Any]:
        result = {}
        if self.search_types is not None:
            result["searchType"] = [
                st.value if isinstance(st, Enum) else st
                for st in self.search_types
            ]
        if self.dense_vector_search_configuration is not None:
            result["denseVectorSearchConfiguration"] = self.dense_vector_search_configuration.to_dict()
        if self.fulltext_search_configuration is not None:
            result["fullTextSearchConfiguration"] = self.fulltext_search_configuration.to_dict()
        if self.reranking_configuration is not None:
            result["rerankingConfiguration"] = self.reranking_configuration.to_dict()
        if self.filter is not None:
            result["filter"] = self.filter.to_dict()
        return result

@dataclass
class RetrieveRequest(BaseModel):
    """Request model for retrieving from a knowledge base."""
    knowledge_base_name: str
    retrieval_query: RetrievalQuery
    subspace: Optional[List[str]] = None
    retrieval_configuration: Optional[RetrievalConfiguration] = None

    def to_dict(self) -> Dict[str, Any]:
        result: Dict[str, Any] = {
            "knowledgeBaseName": self.knowledge_base_name,
            "retrievalQuery": self.retrieval_query.to_dict()
        }
        if self.subspace is not None:
            result["subspace"] = self.subspace
        if self.retrieval_configuration is not None:
            result["retrievalConfiguration"] = self.retrieval_configuration.to_dict()
        return result
