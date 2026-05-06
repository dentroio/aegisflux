//! WO-DET-005: controller-driven observe-only detection packs (Windows only).

mod cache;
mod eval;
mod http;
mod pipeline;
mod schema_check;
mod verify;

pub use pipeline::run_dynamic_pack_pipeline;
