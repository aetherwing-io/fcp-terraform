import { describe, it, expect, beforeEach } from "vitest";
import {
  createEmptyConfig, addBlock, removeBlock, findByLabel,
  findByType, findByKind, findByProvider, findConnections,
  addConnection, removeConnection, createBlock, deriveProvider,
  rebuildLabelIndex, generateId,
} from "../src/model.js";
import type { TerraformConfig, Connection } from "../src/types.js";

describe("deriveProvider", () => {
  it("extracts aws", () => expect(deriveProvider("aws_s3_bucket")).toBe("aws"));
  it("extracts google", () => expect(deriveProvider("google_compute_instance")).toBe("google"));
  it("extracts azurerm", () => expect(deriveProvider("azurerm_resource_group")).toBe("azurerm"));
  it("handles no underscore", () => expect(deriveProvider("random")).toBe("random"));
});

describe("ConfigModel", () => {
  let config: TerraformConfig;

  beforeEach(() => {
    config = createEmptyConfig("Test");
    rebuildLabelIndex(config);
  });

  describe("addBlock", () => {
    it("adds a block", () => {
      const block = createBlock("resource", "aws_s3_bucket", "assets", { acl: "private" });
      const err = addBlock(config, block);
      expect(err).toBeNull();
      expect(config.blocks.size).toBe(1);
      expect(config.blockOrder).toContain(block.id);
    });

    it("allows same label on different types", () => {
      addBlock(config, createBlock("resource", "aws_s3_bucket", "assets", {}));
      const err = addBlock(config, createBlock("resource", "aws_instance", "assets", {}));
      expect(err).toBeNull();
      expect(config.blocks.size).toBe(2);
    });

    it("enforces uniqueness for same type+label", () => {
      addBlock(config, createBlock("resource", "aws_s3_bucket", "assets", {}));
      const err = addBlock(config, createBlock("resource", "aws_s3_bucket", "assets", {}));
      expect(err).toContain("already exists");
    });

    it("is case-insensitive for type+label uniqueness", () => {
      addBlock(config, createBlock("resource", "aws_s3_bucket", "Assets", {}));
      const err = addBlock(config, createBlock("resource", "aws_s3_bucket", "assets", {}));
      expect(err).toContain("already exists");
    });
  });

  describe("findByLabel", () => {
    it("finds by label", () => {
      const block = createBlock("resource", "aws_s3_bucket", "mybucket", {});
      addBlock(config, block);
      expect(findByLabel(config, "mybucket")).toBe(block);
    });

    it("finds case-insensitively", () => {
      const block = createBlock("resource", "aws_s3_bucket", "MyBucket", {});
      addBlock(config, block);
      expect(findByLabel(config, "mybucket")).toBe(block);
    });

    it("returns undefined for missing", () => {
      expect(findByLabel(config, "nonexistent")).toBeUndefined();
    });

    it("returns undefined for ambiguous label", () => {
      addBlock(config, createBlock("resource", "aws_vpc", "main", {}));
      addBlock(config, createBlock("resource", "aws_internet_gateway", "main", {}));
      expect(findByLabel(config, "main")).toBeUndefined();
    });

    it("resolves qualified label for ambiguous cases", () => {
      const vpc = createBlock("resource", "aws_vpc", "main", { cidr_block: "10.0.0.0/16" });
      const igw = createBlock("resource", "aws_internet_gateway", "main", {});
      addBlock(config, vpc);
      addBlock(config, igw);
      expect(findByLabel(config, "aws_vpc.main")).toBe(vpc);
      expect(findByLabel(config, "aws_internet_gateway.main")).toBe(igw);
    });
  });

  describe("removeBlock", () => {
    it("removes a block", () => {
      const block = createBlock("resource", "aws_s3_bucket", "assets", {});
      addBlock(config, block);
      const removed = removeBlock(config, block.id);
      expect(removed).toBe(block);
      expect(config.blocks.size).toBe(0);
      expect(findByLabel(config, "assets")).toBeUndefined();
    });

    it("removes associated connections", () => {
      const b1 = createBlock("resource", "aws_instance", "web", {});
      const b2 = createBlock("resource", "aws_s3_bucket", "assets", {});
      addBlock(config, b1);
      addBlock(config, b2);
      const conn: Connection = {
        id: generateId(), sourceId: b1.id, targetId: b2.id,
        sourceLabel: "web", targetLabel: "assets",
      };
      addConnection(config, conn);
      removeBlock(config, b1.id);
      expect(config.connections.size).toBe(0);
    });
  });

  describe("findByType/Kind/Provider", () => {
    beforeEach(() => {
      addBlock(config, createBlock("resource", "aws_s3_bucket", "b1", {}));
      addBlock(config, createBlock("resource", "aws_instance", "i1", {}));
      addBlock(config, createBlock("variable", "variable", "env", {}));
    });

    it("findByType", () => {
      expect(findByType(config, "aws_s3_bucket")).toHaveLength(1);
    });

    it("findByKind", () => {
      expect(findByKind(config, "resource")).toHaveLength(2);
      expect(findByKind(config, "variable")).toHaveLength(1);
    });

    it("findByProvider", () => {
      expect(findByProvider(config, "aws")).toHaveLength(2);
    });
  });

  describe("connections", () => {
    it("adds and finds connections", () => {
      const b1 = createBlock("resource", "aws_instance", "web", {});
      const b2 = createBlock("resource", "aws_rds_instance", "db", {});
      addBlock(config, b1);
      addBlock(config, b2);

      const conn: Connection = {
        id: generateId(), sourceId: b1.id, targetId: b2.id,
        sourceLabel: "web", targetLabel: "db",
      };
      addConnection(config, conn);
      expect(findConnections(config, b1.id)).toHaveLength(1);
      expect(findConnections(config, b2.id)).toHaveLength(1);
    });
  });
});
