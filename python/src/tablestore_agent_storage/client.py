"""Agent Storage client implementation."""

import os
from copy import deepcopy
from typing import Dict, List, Optional, Tuple, Union
import oss2
import tablestore
from tablestore import OTSServiceError

from tablestore_agent_storage.models import (
    CreateKnowledgeBaseRequest,
    DeleteKnowledgeBaseRequest,
    DescribeKnowledgeBaseRequest,
    ListKnowledgeBaseRequest,
    UpdateKnowledgeBaseRequest,
    AddDocumentsRequest,
    UploadDocumentsRequest,
    UpdateDocumentRequest,
    GetDocumentRequest,
    ListDocumentsRequest,
    DeleteDocumentsRequest,
    RetrieveRequest,
    ListChunksRequest,
    UpdateChunksRequest,
)


# Constants
DEFAULT_SUBSPACE = "_default"
OSS_PROTOCOL_PREFIX = "oss://"
OSS_PATH_SEPARATOR = "/"
SOURCE_DOC_FOLDER = "source_doc"

# Error messages
ERROR_OSS_NOT_INITIALIZED = "OSS client not initialized. Please provide OSS configuration."
ERROR_OTS_NOT_INITIALIZED = "OTS client not initialized. Please provide OTS configuration."
ERROR_MISSING_KNOWLEDGE_BASE_NAME = "Request must contain 'knowledgeBaseName' field"
ERROR_MISSING_DOCUMENTS = "Request must contain 'documents' field"
ERROR_DOCUMENTS_NOT_LIST = "'documents' field must be a list"
ERROR_MISSING_FILE_PATH = "Each document must contain 'filePath' field"


def _to_dict(request: Union[Dict, object]) -> Dict:
    """Convert request to dictionary if it has to_dict method."""
    if hasattr(request, 'to_dict'):
        return request.to_dict()
    return request


class AgentStorageClient:
    """Main storage client for OSS and OTS operations with knowledge base support."""

    def __init__(self, access_key_id=None, access_key_secret=None, sts_token=None, oss_endpoint=None,
                 oss_bucket_name=None, ots_endpoint=None, ots_instance_name=None, **kwargs):
        """
        Initialize storage client with unified credentials.

        Args:
            access_key_id: Aliyun access key ID (used for both OSS and OTS)
            access_key_secret: Aliyun access key secret (used for both OSS and OTS)
            oss_endpoint: OSS endpoint URL
            oss_bucket_name: OSS bucket name
            ots_endpoint: OTS endpoint URL
            ots_instance_name: OTS instance name
        """
        self.access_key_id = access_key_id
        self.access_key_secret = access_key_secret
        self.sts_token = sts_token
        self.oss_endpoint = oss_endpoint
        self.oss_bucket_name = oss_bucket_name
        self.ots_endpoint = ots_endpoint
        self.ots_instance_name = ots_instance_name

        self.oss_client = None
        self.ots_client = None

        if access_key_id and access_key_secret:
            if oss_endpoint and oss_bucket_name:
                self._init_oss()
            if ots_endpoint and ots_instance_name:
                self._init_ots(**kwargs)
            else:
                raise ValueError(ERROR_OTS_NOT_INITIALIZED)
        else:
            raise ValueError(ERROR_OTS_NOT_INITIALIZED)

    def _init_oss(self):
        """Initialize OSS client."""
        if not self.sts_token:
            auth = oss2.Auth(self.access_key_id, self.access_key_secret)
            self.oss_client = oss2.Bucket(auth, self.oss_endpoint, self.oss_bucket_name)
        else:
            auth = oss2.StsAuth(self.access_key_id, self.access_key_secret, self.sts_token)
            self.oss_client = oss2.Bucket(auth, self.oss_endpoint, self.oss_bucket_name)

    def _init_ots(self, **kwargs):
        """Initialize OTS client."""
        kwargs['sts_token'] = self.sts_token
        self.ots_client = tablestore.OTSClient(
            self.ots_endpoint,
            self.access_key_id,
            self.access_key_secret,
            self.ots_instance_name,
            **kwargs
        )

    def _validate_oss_client(self):
        """Validate that OSS client is initialized."""
        if not self.oss_client:
            raise ValueError(ERROR_OSS_NOT_INITIALIZED)

    def _validate_ots_client(self):
        """Validate that OTS client is initialized."""
        if not self.ots_client:
            raise ValueError(ERROR_OTS_NOT_INITIALIZED)

    def _validate_request_fields(self, request: Dict, required_fields: List[str]):
        """
        Validate that required fields exist in request.
        
        Args:
            request: Request dictionary to validate
            required_fields: List of required field names
            
        Raises:
            ValueError: If any required field is missing
        """
        for field in required_fields:
            if field not in request:
                raise ValueError(f"Request must contain '{field}' field")

    def _validate_documents_field(self, request: Dict):
        """
        Validate documents field in request.
        
        Args:
            request: Request dictionary containing documents field
            
        Raises:
            ValueError: If documents field is missing or not a list
        """
        if 'documents' not in request:
            raise ValueError(ERROR_MISSING_DOCUMENTS)
        if not isinstance(request['documents'], list):
            raise ValueError(ERROR_DOCUMENTS_NOT_LIST)

    def _get_subspace(self, request: Dict) -> str:
        """
        Get subspace from request or return default.
        
        Args:
            request: Request dictionary
            
        Returns:
            Subspace name
        """
        return request.get('subspace', DEFAULT_SUBSPACE)

    def _build_oss_key(self, kb_name: str, subspace: str, file_path: str) -> str:
        """
        Build OSS key path for uploaded file.
        
        Args:
            kb_name: Knowledge base name
            subspace: Subspace name
            file_path: Local file path
            
        Returns:
            OSS key path
        """
        abs_path = os.path.abspath(file_path).lstrip(OSS_PATH_SEPARATOR)
        return f"{self.ots_instance_name}{OSS_PATH_SEPARATOR}{kb_name}{OSS_PATH_SEPARATOR}{SOURCE_DOC_FOLDER}{OSS_PATH_SEPARATOR}{subspace}{OSS_PATH_SEPARATOR}{abs_path}"

    def _build_oss_full_path(self, oss_key: str) -> str:
        """
        Build full OSS path with protocol and bucket.
        
        Args:
            oss_key: OSS object key
            
        Returns:
            Full OSS path (oss://bucket/key)
        """
        return f"{OSS_PROTOCOL_PREFIX}{self.oss_bucket_name}{OSS_PATH_SEPARATOR}{oss_key}"

    def _parse_oss_key(self, oss_path: str) -> Optional[Tuple[str, str]]:
        """
        Parse OSS path to extract bucket name and object key.
        
        Args:
            oss_path: Full OSS path (oss://bucket/key or just key)
            
        Returns:
            Tuple of (bucket_name, object_key) or None if invalid format
        """
        if oss_path.startswith(OSS_PROTOCOL_PREFIX):
            # Remove 'oss://' prefix
            path_parts = oss_path[len(OSS_PROTOCOL_PREFIX):].split(OSS_PATH_SEPARATOR, 1)
            if len(path_parts) == 2:
                return path_parts[0], path_parts[1]
            return None
        else:
            # Assume it's a direct object key without oss:// prefix
            return self.oss_bucket_name, oss_path

    def create_knowledge_base(self, request: Union[Dict, CreateKnowledgeBaseRequest]):
        """
        Create a new knowledge base.

        Args:
            request: Dict or CreateKnowledgeBaseRequest containing knowledge base configuration
            Example using Model:
                CreateKnowledgeBaseRequest(
                    knowledge_base_name="test_kb",
                    description="xxx yyy",
                    subspace=True,
                    tags=["tag_name"],
                    metadata=[MetadataField(name="author", type=MetadataFieldType.STRING)]
                )
            Example using Dict:
                {
                  "knowledgeBaseName": "test_kb",
                  "description": "xxx yyy",
                  "subspace": True,
                  "tags": ["tag_name"],
                  "metadata": [{"name": "author", "type": "string"}]
                }

        Returns:
            Response from API
            Example:
            {
              "code": "SUCCESS",
              "data": {},
              "message": "succeed",
              "requestId": "xxx"
            }

        Raises:
            OTSServiceError: If OTS request to server fails
            OTSClientError: If OTS request fails with client-side error
        """
        return self.ots_client.create_knowledge_base(_to_dict(request))

    def delete_knowledge_base(self, request: Union[Dict, DeleteKnowledgeBaseRequest]):
        """
        Delete an existing knowledge base.

        Args:
            request: Dict or DeleteKnowledgeBaseRequest containing knowledge base identifier
            Example using Model:
                DeleteKnowledgeBaseRequest(knowledge_base_name="test_kb")
            Example using Dict:
                {"knowledgeBaseName": "test_kb"}

        Returns:
            Response from API
            Example:
            {
              "code": "SUCCESS",
              "data": {},
              "message": "succeed",
              "requestId": "xxx"
            }

        Raises:
            OTSServiceError: If OTS request to server fails
            OTSClientError: If OTS request fails with client-side error
        """
        return self.ots_client.delete_knowledge_base(_to_dict(request))

    def describe_knowledge_base(self, request: Union[Dict, DescribeKnowledgeBaseRequest]):
        """
        Get details of a knowledge base.

        Args:
            request: Dict or DescribeKnowledgeBaseRequest containing knowledge base identifier
            Example using Model:
                DescribeKnowledgeBaseRequest(knowledge_base_name="test_kb")
            Example using Dict:
                {"knowledgeBaseName": "test_kb"}

        Returns:
            Response from API with knowledge base details
            Example:
            {
                "code": "SUCCESS",
                "data": {
                    "knowledgeBaseName": "test_kb",
                    "description": "xxx yyy",
                    "subspace": True,
                    "tags": ["tag_name"],
                    "metadata": [{"name": "author", "type": "string"}],
                    "createdAt": 1769051055111,
                    "updatedAt": 1769051061346
                },
                "message": "succeed",
                "requestId": "xxx"
            }

        Raises:
            OTSServiceError: If OTS request to server fails
            OTSClientError: If OTS request fails with client-side error
        """
        return self.ots_client.describe_knowledge_base(_to_dict(request))

    def list_knowledge_base(self, request: Union[Dict, ListKnowledgeBaseRequest] = None):
        """
        List all knowledge bases.

        Args:
            request: Dict or ListKnowledgeBaseRequest containing list parameters (e.g., pagination)
            Example using Model:
                ListKnowledgeBaseRequest(max_results=10, next_token="xxx")
            Example using Dict:
                {"maxResults": 10, "nextToken": "xxx"}

        Returns:
            Response from API with list of knowledge bases
            Example:
            {
                "code": "SUCCESS",
                "data": {
                    "knowledgeBases": [{
                        "knowledgeBaseName": "test_kb",
                        "description": "",
                        "subspace": true,
                        "tags": ["tag_name"],
                        "createdAt": 1769051256049,
                        "updatedAt": 1769051256049
                    }],
                    "nextToken": ""
                },
                "message": "succeed",
                "requestId": "xxx"
            }

        Raises:
            OTSServiceError: If OTS request to server fails
            OTSClientError: If OTS request fails with client-side error
        """
        if request is None:
            request = {}
        return self.ots_client.list_knowledge_base(_to_dict(request))

    def update_knowledge_base(self, request: Union[Dict, UpdateKnowledgeBaseRequest]):
        """
        Update a knowledge base.

        Args:
            request: Dict or UpdateKnowledgeBaseRequest containing update parameters
            Example using Model:
                UpdateKnowledgeBaseRequest(
                    knowledge_base_name="test_kb",
                    description="updated description",
                    tags=["new_tag"]
                )
            Example using Dict:
                {"knowledgeBaseName": "test_kb", "description": "updated description", "tags": ["new_tag"]}

        Returns:
            Response from API

        Raises:
            OTSServiceError: If OTS request to server fails
            OTSClientError: If OTS request fails with client-side error
        """
        return self.ots_client.update_knowledge_base(_to_dict(request))

    def add_documents(self, request: Union[Dict, AddDocumentsRequest]):
        """
        Add a document to a knowledge base.

        Args:
            request: Dict or AddDocumentsRequest containing document data and knowledge base identifier
                Example using Model:
                    AddDocumentsRequest(
                        knowledge_base_name="test_kb",
                        subspace="xxx",
                        documents=[
                            AddDocumentItem(oss_key="oss://bucketname/yyy/zzz.pdf", metadata={"author": "aliyun"}),
                            AddDocumentItem(oss_key="oss://bucketname/bbb/ccc.docx", metadata={"author": "aliyun"})
                        ]
                    )
                Example using Dict:
                    {
                        "knowledgeBaseName": "test_kb",
                        "subspace": "xxx",
                        "documents": [
                            {"ossKey": "oss://bucketname/yyy/zzz.pdf", "metadata": {"author": "aliyun"}},
                            {"ossKey": "oss://bucketname/bbb/ccc.docx", "metadata": {"author": "aliyun"}}
                        ]
                    }

        Returns:
            Response from API
            Example:
            {
                "code": "SUCCESS",
                "data": {
                    "documentDetails": [
                        {"docId": "fc6ed97f-a036-489f-ba79-79e4c766d3af", "status": "succeed", "ossKey": "oss://bucketname/yyy/zzz.pdf"},
                        {"docId": "940f2c5c-fdc0-4b4e-8bd1-5b47fa30c0d3", "status": "succeed", "ossKey": "oss://bucketname/bbb/ccc.docx"}
                    ]
                },
                "message": "succeed",
                "requestId": "xxx"
            }

        Raises:
            OTSServiceError: If OTS request to server fails
            OTSClientError: If OTS request fails with client-side error
        """
        return self.ots_client.add_documents(_to_dict(request))

    def upload_documents(self, request: Union[Dict, UploadDocumentsRequest]):
        """
        Add a document to a knowledge base by uploading a local file to OSS first.

        This method differs from add_documents in that it accepts a local file path
        instead of an OSS key. It will:
        1. Upload the file to OSS
        2. Replace the file path in the request with the OSS path
        3. Call the OTS add_documents interface

        Args:
            request: Dict or UploadDocumentsRequest containing document data and knowledge base identifier
                Example using Model:
                    UploadDocumentsRequest(
                        knowledge_base_name="test_kb",
                        subspace="xxx",
                        documents=[
                            UploadDocumentItem(file_path="/path/to/local/file.pdf", metadata={"author": "aliyun"})
                        ]
                    )
                Example using Dict:
                    {
                        "knowledgeBaseName": "test_kb",
                        "subspace": "xxx",
                        "documents": [{"filePath": "/path/to/local/file.pdf", "metadata": {"author": "aliyun"}}]
                    }

        Returns:
            Response from API
            Example:
            {
                "code": "SUCCESS",
                "data": {
                    "documentDetails": [{
                        "docId": "fc6ed97f-a036-489f-ba79-79e4c766d3af",
                        "status": "succeed",
                        "ossKey": "oss://bucketname/instancename/knowledgebase_name/subspace_name/uploads/abs_path//file.pdf"
                    }]
                },
                "message": "succeed",
                "requestId": "xxx"
            }

        Raises:
            ValueError: If OSS client is not initialized or file doesn't exist
            FileNotFoundError: If the specified file path doesn't exist
            OTSServiceError: If OTS request to server fails
            OTSClientError: If OTS request fails with client-side error
        """
        self._validate_oss_client()
        self._validate_ots_client()

        # Convert to dict if needed and deep copy to avoid modifying the original
        modified_request = deepcopy(_to_dict(request))

        # Validate required fields
        self._validate_request_fields(modified_request, ['knowledgeBaseName'])
        self._validate_documents_field(modified_request)

        kb_name = modified_request['knowledgeBaseName']
        subspace = self._get_subspace(modified_request)

        # Process each document in the request
        oss_keys = []
        for doc in modified_request['documents']:
            if 'filePath' not in doc:
                raise ValueError(ERROR_MISSING_FILE_PATH)

            file_path = doc['filePath']

            # Check if file exists
            if not os.path.exists(file_path):
                raise FileNotFoundError(f"File not found: {file_path}")

            # Build OSS key and upload file
            oss_key = self._build_oss_key(kb_name, subspace, file_path)
            oss_keys.append(oss_key)
            
            try:
                # Upload file to OSS
                with open(file_path, 'rb') as f:
                    self.oss_client.put_object(oss_key, f)
            except Exception as e:
                raise ValueError(f"Failed to upload file to OSS: {e}")

            # Replace filePath with ossKey
            del doc['filePath']
            doc['ossKey'] = self._build_oss_full_path(oss_key)

        # Call the original add_documents method with modified request
        try:
            add_documents_reps = self.ots_client.add_documents(modified_request)
            return add_documents_reps
        except OTSServiceError as e:
            if oss_keys:
                self.oss_client.batch_delete_objects(oss_keys)
            raise e

    def update_document(self, request: Union[Dict, UpdateDocumentRequest]):
        """
        Update a document's metadata in a knowledge base.

        Args:
            request: Dict or UpdateDocumentRequest containing document update parameters
            Example using Model:
                UpdateDocumentRequest(
                    knowledge_base_name="test_kb",
                    subspace="xxx",
                    doc_id="doc_123",
                    metadata={"author": "updated_author"}
                )
            Example using Dict:
                {"knowledgeBaseName": "test_kb", "subspace": "xxx", "docId": "doc_123", "metadata": {"author": "updated_author"}}

        Returns:
            Response from API

        Raises:
            OTSServiceError: If OTS request to server fails
            OTSClientError: If OTS request fails with client-side error
        """
        return self.ots_client.update_document(_to_dict(request))

    def get_document(self, request: Union[Dict, GetDocumentRequest]):
        """
        Get a document from a knowledge base.

        Args:
            request: Dict or GetDocumentRequest containing document identifier
            Example using Model:
                GetDocumentRequest(knowledge_base_name="test_kb", subspace="xxx", doc_id="doc_123")
                or
                GetDocumentRequest(knowledge_base_name="test_kb", subspace="xxx", oss_key="oss://bucket/file.pdf")
            Example using Dict:
                {"knowledgeBaseName": "test_kb", "subspace": "xxx", "docId": "doc_123"}
                or
                {"knowledgeBaseName": "test_kb", "subspace": "xxx", "ossKey": "oss://bucket/file.pdf"}

        Returns:
            Response from API with document data
            Examples:
            {
                "code": "SUCCESS",
                "data": [{
                    "docId": "12baf53e-de36-4168-85d4-6de531fc9097",
                    "ossKey": "oss://bucketname/xxx.pdf",
                    "subspace": "test",
                    "status": "pending",
                    "createdAt": 1769052421132,
                    "eTag": "C3988BE8AAEDAE172ED41F851D5F40B4",
                    "updatedAt": 1769052421132,
                    "metadata": {"date": "2026-01-22 10:00:59", "double": 3.14, "long": 314}
                }],
                "message": "succeed",
                "requestId": "xxx"
            }

        Raises:
            OTSServiceError: If OTS request to server fails
            OTSClientError: If OTS request fails with client-side error
        """
        return self.ots_client.get_document(_to_dict(request))

    def list_documents(self, request: Union[Dict, ListDocumentsRequest]):
        """
        List documents in a knowledge base.

        Args:
            request: Dict or ListDocumentsRequest containing knowledge base identifier and list parameters
            Example using Model:
                ListDocumentsRequest(knowledge_base_name="test_kb", subspace="xxx", max_results=10)
            Example using Dict:
                {"knowledgeBaseName": "test_kb", "subspace": "xxx", "maxResults": 10}

        Returns:
            Response from API with list of documents
            Examples:
            {
                "code": "SUCCESS",
                "data": {
                    "documents": [{
                        "docId": "940f2c5c-fdc0-4b4e-8bd1-5b47fa30c0d3",
                        "ossKey": "oss://bucketname/xxx.pdf",
                        "subspace": "test",
                        "chunkNum": 808,
                        "status": "completed",
                        "createdAt": 1769051903177,
                        "eTag": "73940BDD1496584E6E5C6F7ECDBC7DFB",
                        "updatedAt": 1769051903177
                    }],
                    "nextToken": ""
                },
                "message": "succeed",
                "requestId": "xxx"
            }

        Raises:
            OTSServiceError: If OTS request to server fails
            OTSClientError: If OTS request fails with client-side error
        """
        return self.ots_client.list_documents(_to_dict(request))

    def delete_documents(self, request: Union[Dict, DeleteDocumentsRequest], delete_file: bool = False):
        """
        Delete a document from a knowledge base.

        This method will:
        1. Delete the document metadata from OTS
        2. If delete_file, delete the actual file from OSS

        Args:
            request: Dict or DeleteDocumentsRequest containing document identifier
            Example using Model:
                DeleteDocumentsRequest(
                    knowledge_base_name="test_kb",
                    subspace="xxx",
                    documents=[DeleteDocumentItem(doc_id="doc_123")]
                )
                or
                DeleteDocumentsRequest(
                    knowledge_base_name="test_kb",
                    subspace="xxx",
                    documents=[DeleteDocumentItem(oss_key="oss://bucket/path/to/file.pdf")]
                )
            Example using Dict:
                {"knowledgeBaseName": "test_kb", "subspace": "xxx", "documents": [{"docId": "doc_123"}]}
                or
                {"knowledgeBaseName": "test_kb", "subspace": "xxx", "documents": [{"ossKey": "oss://bucket/path/to/file.pdf"}]}
            delete_file: whether to delete the file from OSS

        Returns:
            Response from API
            Example:
            {
                "code": "SUCCESS",
                "data": {
                    "documentDetails": [
                        {"status": "succeed", "ossKey": "oss://testbucket/xxx.pdf"},
                        {"status": "succeed", "ossKey": "oss://testbucket/yyy.pdf"}
                    ]
                },
                "message": "succeed",
                "requestId": "xxx"
            }

        Raises:
            ValueError: If required fields are missing or clients not initialized
            OTSServiceError: If OTS request to server fails
            OTSClientError: If OTS request fails with client-side error
        """
        self._validate_ots_client()

        if delete_file:
            self._validate_oss_client()

        # Convert to dict if needed
        request_dict = _to_dict(request)

        # Validate request
        self._validate_request_fields(request_dict, ['knowledgeBaseName'])
        self._validate_documents_field(request_dict)

        kb_name = request_dict['knowledgeBaseName']
        subspace = self._get_subspace(request_dict)

        # Collect OSS keys to delete
        oss_keys_to_delete = []

        if delete_file:
            oss_keys_to_delete = self._collect_oss_keys_for_deletion(request_dict, kb_name, subspace)

        # Delete document metadata from OTS
        delete_response = self.ots_client.delete_documents(request_dict)

        # Delete actual files from OSS
        if delete_file and oss_keys_to_delete:
            self._delete_oss_files(oss_keys_to_delete)

        return delete_response

    def _collect_oss_keys_for_deletion(self, request: Dict, kb_name: str, subspace: str) -> List[str]:
        """
        Collect OSS keys from documents for deletion.
        
        Args:
            request: Request containing documents
            kb_name: Knowledge base name
            subspace: Subspace name
            
        Returns:
            List of OSS keys to delete
        """
        oss_keys = []
        
        for doc in request['documents']:
            oss_key = None

            # If docId is provided, get the document first to retrieve ossKey
            if 'docId' in doc:
                oss_key = self._get_oss_key_by_doc_id(doc['docId'], kb_name, subspace)
            # If ossKey is directly provided in the request
            elif 'ossKey' in doc:
                oss_key = doc['ossKey']

            # Store ossKey for later deletion from OSS
            if oss_key:
                oss_keys.append(oss_key)
                
        return oss_keys

    def _get_oss_key_by_doc_id(self, doc_id: str, kb_name: str, subspace: str) -> Optional[str]:
        """
        Get OSS key for a document by its ID.
        
        Args:
            doc_id: Document ID
            kb_name: Knowledge base name
            subspace: Subspace name
            
        Returns:
            OSS key or None if not found
        """
        get_request = {
            'knowledgeBaseName': kb_name,
            'subspace': subspace,
            'docId': doc_id
        }

        try:
            doc_infos = self.ots_client.get_document(get_request)
            if doc_infos.get("code") != "SUCCESS":
                raise Exception(f"Failed to get document {doc_id}, response: {doc_infos}")
            
            data = doc_infos.get("data", [])
            if not data:
                return None

            # Extract ossKey from response
            for doc_info in data:
                if doc_info.get("ossKey"):
                    return doc_info["ossKey"]

        except Exception as e:
            # Log warning but continue with deletion
            self.ots_client.logger.warning(f"Warning: Failed to get document {doc_id}: {e}")
            
        return None

    def _delete_oss_files(self, oss_keys: List[str]):
        """
        Delete files from OSS.
        
        Args:
            oss_keys: List of OSS keys to delete
        """
        keys_to_delete = []
        
        for oss_key in oss_keys:
            parsed = self._parse_oss_key(oss_key)
            if parsed:
                bucket_name, object_key = parsed
                
                # Only delete if it's from the current bucket
                if bucket_name == self.oss_bucket_name:
                    keys_to_delete.append(object_key)
                else:
                    self.ots_client.logger.warning(
                        f"Warning: OSS key {oss_key} is from different bucket, skipping deletion")
            else:
                self.ots_client.logger.warning(f"Warning: Invalid OSS key format: {oss_key}")

        # Batch delete objects
        if keys_to_delete:
            try:
                self.oss_client.batch_delete_objects(keys_to_delete)
            except Exception as e:
                # Log error but don't fail the entire operation
                self.ots_client.logger.warning(f"Warning: Failed to delete OSS objects {keys_to_delete}: {e}")

    def retrieve(self, request: Union[Dict, RetrieveRequest]):
        """
        Perform vector retrieval/search in a knowledge base.

        Args:
            request: Dict or RetrieveRequest containing search query and parameters
            Example using Model:
                RetrieveRequest(
                    knowledge_base_name="test_kb",
                    subspace=["xxx"],
                    retrieval_query=RetrievalQuery(text="阿里云", type=RetrievalQueryType.TEXT),
                    retrieval_configuration=RetrievalConfiguration(
                        search_types=[SearchType.DENSE_VECTOR, SearchType.TEXT],
                        dense_vector_search_configuration=DenseVectorSearchConfiguration(number_of_results=10),
                        fulltext_search_configuration=FulltextSearchConfiguration(number_of_results=10),
                        reranking_configuration=RerankingConfiguration(
                            type=RerankingType.RRF,
                            number_of_reranked_results=10,
                            rrf_configuration=RRFRerankingConfiguration(
                                dense_vector_search_weight=1.0,
                                text_search_weight=1.0,
                            )
                        ),
                        filter=EqualsFilter(key="category", value="cloud")
                    )
                )
            Example using Dict:
                {
                    "knowledgeBaseName": "test_kb",
                    "subspace": ["xxx"],
                    "retrievalQuery": {"text": "阿里云", "type": "TEXT"},
                    "retrievalConfiguration": {
                        "searchType": ["DENSE_VECTOR"],
                        "denseVectorSearchConfiguration": {"numberOfResults": 10},
                        "rerankingConfiguration": {
                            "type": "RRF",
                            "numberOfRerankedResults": 10,
                            "rrfConfiguration": {"denseVectorSearchWeight": 1, "textSearchWeight": 1, "k": 60}
                        }
                    }
                }

        Returns:
            Response from API with search results
            Example:
            {
                "code": "SUCCESS",
                "data": {
                    "retrievalResults": [{
                        "ossKey": "oss://testbucket/xxx.pdf",
                        "docId": "96fb386e-44d5-40aa-aa4d-edc0762f867c",
                        "chunkId": 3,
                        "subspace": "test",
                        "score": 0.1,
                        "content": "xxxx",
                        "metadata": {"date": "2026-01-22 10:00:59", "double": 3.14, "long": 314}
                    }]
                },
                "message": "success"
            }

        Raise:
            OTSServiceError: If OTS request to server fails
            OTSClientError: If OTS request fails with client-side error
        """
        return self.ots_client.retrieve(_to_dict(request))

    def list_chunks(self, request: Union[Dict, ListChunksRequest]):
        """
        List chunks in a knowledge base.

        Args:
            request: Dict or ListChunksRequest containing chunk list parameters
            Example using Model:
                ListChunksRequest(
                    knowledge_base_name="test_kb",
                    doc_id="doc_123",
                    max_results=10
                )
            Example using Dict:
                {
                    "knowledgeBaseName": "test_kb",
                    "docId": "doc_123",
                    "maxResults": 10
                }

        Returns:
            Response from API with chunk list
            Example:
            {
                "code": "SUCCESS",
                "data": {
                    "chunkDetails": [{
                        "docId": "doc_123",
                        "ossKey": "oss://bucket/file.pdf",
                        "chunkId": 1,
                        "content": "chunk content text",
                        "title": "chunk title",
                        "chunkType": "text",
                        "status": "active",
                        "createdAt": 1769052421132,
                        "updatedAt": 1769052421132
                    }],
                    "nextToken": ""
                },
                "message": "succeed",
                "requestId": "xxx"
            }

        Raises:
            OTSServiceError: If OTS request to server fails
            OTSClientError: If OTS request fails with client-side error
        """
        return self.ots_client.list_chunks(_to_dict(request))

    def update_chunks(self, request: Union[Dict, UpdateChunksRequest]):
        """
        Update chunks in a knowledge base.

        Args:
            request: Dict or UpdateChunksRequest containing chunk update parameters
            Example using Model:
                UpdateChunksRequest(
                    knowledge_base_name="test_kb",
                    chunks=[
                        UpdateChunkItem(chunk_id=1, doc_id="doc_123", content="new content"),
                        UpdateChunkItem(chunk_id=2, doc_id="doc_123", status="inactive")
                    ]
                )
            Example using Dict:
                {
                    "knowledgeBaseName": "test_kb",
                    "chunks": [
                        {"chunkId": 1, "docId": "doc_123", "content": "new content"},
                        {"chunkId": 2, "docId": "doc_123", "status": "inactive"}
                    ]
                }

        Returns:
            Response from API with update results
            Example:
            {
                "code": "SUCCESS",
                "data": {
                    "updateDetails": [{
                        "docId": "doc_123",
                        "ossKey": "oss://bucket/file.pdf",
                        "chunkId": 1,
                        "updateStatus": "success"
                    }]
                },
                "message": "succeed",
                "requestId": "xxx"
            }

        Raises:
            OTSServiceError: If OTS request to server fails
            OTSClientError: If OTS request fails with client-side error
        """
        return self.ots_client.update_chunks(_to_dict(request))
