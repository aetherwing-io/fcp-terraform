import type { TerraformConfig, TfBlock, Connection, Attribute, NestedBlock } from "./types.js";

let idCounter = 0;

export function generateId(): string {
  return `tf_${(++idCounter).toString(36).padStart(4, "0")}_${Math.random().toString(36).slice(2, 6)}`;
}

/**
 * Derive provider name from a Terraform resource type.
 * "aws_s3_bucket" → "aws", "google_compute_instance" → "google", "azurerm_resource_group" → "azurerm"
 */
export function deriveProvider(fullType: string): string {
  const idx = fullType.indexOf("_");
  return idx > 0 ? fullType.slice(0, idx) : fullType;
}

/**
 * Create a new empty TerraformConfig.
 */
export function createEmptyConfig(title: string): TerraformConfig {
  return {
    id: generateId(),
    title,
    filePath: null,
    blocks: new Map(),
    connections: new Map(),
    blockOrder: [],
  };
}

// ── Label index for O(1) lookup ─────────────────────────

const labelIndex = new Map<string, string[]>(); // label → block IDs

export function rebuildLabelIndex(config: TerraformConfig): void {
  labelIndex.clear();
  for (const [id, block] of config.blocks) {
    const key = block.label.toLowerCase();
    const ids = labelIndex.get(key);
    if (ids) {
      ids.push(id);
    } else {
      labelIndex.set(key, [id]);
    }
  }
}

/**
 * Find a block by label. Returns the block if the label is unambiguous
 * (only one block has that label). Returns undefined if no match or ambiguous.
 */
export function findByLabel(config: TerraformConfig, label: string): TfBlock | undefined {
  // Try qualified label first (e.g., "aws_vpc.main")
  const qualified = findByQualifiedLabel(config, label);
  if (qualified) return qualified;

  const ids = labelIndex.get(label.toLowerCase());
  if (!ids || ids.length === 0) return undefined;
  if (ids.length === 1) return config.blocks.get(ids[0]);
  // Ambiguous — multiple blocks share this label
  return undefined;
}

/**
 * Find a block by qualified label: "fullType.label" (e.g., "aws_vpc.main").
 */
export function findByQualifiedLabel(config: TerraformConfig, input: string): TfBlock | undefined {
  const dotIdx = input.indexOf(".");
  if (dotIdx <= 0) return undefined;
  const typePart = input.slice(0, dotIdx);
  const labelPart = input.slice(dotIdx + 1).toLowerCase();
  const ids = labelIndex.get(labelPart);
  if (!ids) return undefined;
  for (const id of ids) {
    const block = config.blocks.get(id);
    if (block && block.fullType === typePart) return block;
  }
  return undefined;
}

export function findByType(config: TerraformConfig, fullType: string): TfBlock[] {
  const results: TfBlock[] = [];
  for (const block of config.blocks.values()) {
    if (block.fullType === fullType) results.push(block);
  }
  return results;
}

export function findByKind(config: TerraformConfig, kind: string): TfBlock[] {
  const results: TfBlock[] = [];
  for (const block of config.blocks.values()) {
    if (block.kind === kind) results.push(block);
  }
  return results;
}

export function findByProvider(config: TerraformConfig, provider: string): TfBlock[] {
  const results: TfBlock[] = [];
  for (const block of config.blocks.values()) {
    if (block.provider === provider) results.push(block);
  }
  return results;
}

export function findConnections(config: TerraformConfig, blockId: string): Connection[] {
  const results: Connection[] = [];
  for (const conn of config.connections.values()) {
    if (conn.sourceId === blockId || conn.targetId === blockId) results.push(conn);
  }
  return results;
}

export function addBlock(config: TerraformConfig, block: TfBlock): string | null {
  // Check uniqueness: reject only if same fullType AND same label
  for (const existing of config.blocks.values()) {
    if (existing.fullType === block.fullType && existing.label.toLowerCase() === block.label.toLowerCase()) {
      return `${block.fullType} "${block.label}" already exists`;
    }
  }
  config.blocks.set(block.id, block);
  config.blockOrder.push(block.id);
  const key = block.label.toLowerCase();
  const ids = labelIndex.get(key);
  if (ids) {
    ids.push(block.id);
  } else {
    labelIndex.set(key, [block.id]);
  }
  return null;
}

export function removeBlock(config: TerraformConfig, id: string): TfBlock | null {
  const block = config.blocks.get(id);
  if (!block) return null;
  config.blocks.delete(id);
  config.blockOrder = config.blockOrder.filter((bid) => bid !== id);
  const key = block.label.toLowerCase();
  const ids = labelIndex.get(key);
  if (ids) {
    const filtered = ids.filter((bid) => bid !== id);
    if (filtered.length === 0) {
      labelIndex.delete(key);
    } else {
      labelIndex.set(key, filtered);
    }
  }
  // Remove connections involving this block
  for (const [connId, conn] of config.connections) {
    if (conn.sourceId === id || conn.targetId === id) {
      config.connections.delete(connId);
    }
  }
  return block;
}

export function addConnection(config: TerraformConfig, conn: Connection): void {
  config.connections.set(conn.id, conn);
}

export function removeConnection(config: TerraformConfig, id: string): Connection | null {
  const conn = config.connections.get(id);
  if (!conn) return null;
  config.connections.delete(id);
  return conn;
}

/**
 * Create an Attribute from a string value, inferring the type.
 * When forceString is true, skip number/bool detection (user explicitly quoted the value).
 */
export function makeAttribute(key: string, value: string, forceString?: boolean): Attribute {
  if (!forceString) {
    if (value === "true" || value === "false") {
      return { key, value, valueType: "bool" };
    }
    if (/^\d+(\.\d+)?$/.test(value)) {
      return { key, value, valueType: "number" };
    }
  }
  if (value.startsWith("[") && value.endsWith("]")) {
    // Parse list elements and quote non-reference/non-numeric/non-bool values
    const inner = value.slice(1, -1).trim();
    if (inner === "") {
      return { key, value: "[]", valueType: "list" };
    }
    const elements = inner.split(",").map((e) => e.trim());
    const quoted = elements.map((elem) => {
      // References: aws_*, var.*, local.*, data.*, module.*, etc.
      if (/^(aws_|google_|azurerm_|var\.|local\.|data\.|module\.)/.test(elem)) return elem;
      // Numbers
      if (/^\d+(\.\d+)?$/.test(elem)) return elem;
      // Bools
      if (elem === "true" || elem === "false") return elem;
      // Already quoted
      if (elem.startsWith('"') && elem.endsWith('"')) return elem;
      // Everything else: quote it
      return `"${elem}"`;
    });
    return { key, value: `[${quoted.join(", ")}]`, valueType: "list" };
  }
  if (value.startsWith("{")) {
    // JSON object (contains ":") → use map type for jsonencode()
    if (value.includes(":")) {
      return { key, value, valueType: "map" };
    }
    return { key, value, valueType: "expression" };
  }
  // Check for Terraform references: aws_xxx.name.attr or var.xxx
  if (/^(aws_|google_|azurerm_|var\.|local\.|data\.|module\.)/.test(value)) {
    return { key, value, valueType: "reference" };
  }
  return { key, value, valueType: "string" };
}

/**
 * Create a TfBlock from components.
 */
export function createBlock(
  kind: TfBlock["kind"],
  fullType: string,
  label: string,
  attrs: Record<string, string>,
  quotedParams?: Set<string>,
): TfBlock {
  const attributes = new Map<string, Attribute>();
  for (const [k, v] of Object.entries(attrs)) {
    attributes.set(k, makeAttribute(k, v, quotedParams?.has(k)));
  }
  return {
    id: generateId(),
    kind,
    label,
    fullType,
    provider: deriveProvider(fullType),
    attributes,
    nestedBlocks: [],
    tags: new Map(),
    meta: { dependsOn: [] },
  };
}
