/**
 * Lab-only stub: WO-VIS-001 / WO-VIS-007 scenario generation.
 * Usage: node sample_node_agent.mjs
 */
const target = process.env.AEGIS_LAB_TARGET_URL || "https://example.com/";
console.error("sample_node_agent: langchain-style marker for lab detection");
const res = await fetch(target, { method: "GET" });
console.log(res.status);
