//! Minimal end-to-end example.
//!
//! Run with real credentials and a writable instance, e.g.:
//!
//! ```bash
//! OTS_ENDPOINT=https://your-instance.cn-beijing.ots.aliyuncs.com \
//! OTS_INSTANCE=your-instance \
//! OTS_ACCESS_KEY_ID=... OTS_ACCESS_KEY_SECRET=... \
//! cargo run --example quickstart
//! ```

use std::env;

use tablestore_agent_storage::{
    models::{CreateKnowledgeBaseRequest, ListKnowledgeBaseRequest},
    AgentStorageClient,
};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let client = AgentStorageClient::builder()
        .endpoint(env::var("OTS_ENDPOINT")?)
        .instance_name(env::var("OTS_INSTANCE")?)
        .access_key_id(env::var("OTS_ACCESS_KEY_ID")?)
        .access_key_secret(env::var("OTS_ACCESS_KEY_SECRET")?)
        .build()?;

    let list = client
        .list_knowledge_base(ListKnowledgeBaseRequest::default())
        .await?;
    println!("knowledge bases: {list:#?}");

    let create = client
        .create_knowledge_base(CreateKnowledgeBaseRequest::new("kb-rust-demo"))
        .await?;
    println!("created: {create:#?}");

    Ok(())
}
