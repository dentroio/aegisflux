//! WO-DET-004: controller-driven observe-only detection packs (Linux only).

mod cache;
mod eval;
mod http;
mod pipeline;
mod schema_check;
mod verify;

pub use pipeline::run_dynamic_pack_pipeline;
