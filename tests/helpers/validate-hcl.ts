import { parse } from "@cdktf/hcl2json";

/**
 * Validate that a string is syntactically valid HCL.
 * Uses HashiCorp's official WASM-based HCL parser.
 * Throws on invalid syntax.
 */
export async function validateHcl(hcl: string): Promise<void> {
  try {
    await parse("validate.tf", hcl);
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : String(err);
    throw new Error(`Invalid HCL:\n${msg}\n\nGenerated HCL:\n${hcl}`);
  }
}
