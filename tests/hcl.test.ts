import { describe, it, expect, beforeEach } from "vitest";
import { serializeToHcl } from "../src/hcl.js";
import {
  createEmptyConfig, addBlock, createBlock, addConnection,
  rebuildLabelIndex, generateId,
} from "../src/model.js";
import type { TerraformConfig, Connection } from "../src/types.js";
import { validateHcl } from "./helpers/validate-hcl.js";

describe("serializeToHcl", () => {
  let config: TerraformConfig;

  beforeEach(() => {
    config = createEmptyConfig("Test");
    rebuildLabelIndex(config);
  });

  it("serializes a provider", async () => {
    const block = createBlock("provider", "aws", "aws", { region: "us-east-1" });
    block.provider = "aws";
    addBlock(config, block);

    const hcl = serializeToHcl(config);
    expect(hcl).toContain('provider "aws"');
    expect(hcl).toContain('region = "us-east-1"');
    await validateHcl(hcl);
  });

  it("serializes a resource with string and number attrs", async () => {
    const block = createBlock("resource", "aws_instance", "web", {
      ami: "ami-0c55b159",
      instance_type: "t2.micro",
    });
    addBlock(config, block);

    const hcl = serializeToHcl(config);
    expect(hcl).toContain('resource "aws_instance" "web"');
    expect(hcl).toContain('ami  = "ami-0c55b159"');
    expect(hcl).toContain('instance_type = "t2.micro"');
    await validateHcl(hcl);
  });

  it("serializes a resource with bool attrs", async () => {
    const block = createBlock("resource", "aws_s3_bucket", "assets", {
      versioning: "true",
    });
    addBlock(config, block);

    const hcl = serializeToHcl(config);
    expect(hcl).toContain("versioning = true");
    await validateHcl(hcl);
  });

  it("serializes a resource with count", async () => {
    const block = createBlock("resource", "aws_instance", "web", {});
    block.meta.count = "2";
    addBlock(config, block);

    const hcl = serializeToHcl(config);
    expect(hcl).toContain("count = 2");
    await validateHcl(hcl);
  });

  it("serializes tags", async () => {
    const block = createBlock("resource", "aws_instance", "web", {});
    block.tags.set("Name", "WebServer");
    block.tags.set("Env", "prod");
    addBlock(config, block);

    const hcl = serializeToHcl(config);
    expect(hcl).toContain("tags = {");
    expect(hcl).toContain('Name = "WebServer"');
    expect(hcl).toContain('Env = "prod"');
    await validateHcl(hcl);
  });

  it("serializes a variable", async () => {
    const block = createBlock("variable", "variable", "env", {
      type: "string",
      default: "production",
    });
    block.provider = "";
    addBlock(config, block);

    const hcl = serializeToHcl(config);
    expect(hcl).toContain('variable "env"');
    expect(hcl).toContain("type    = string");
    expect(hcl).toContain('default = "production"');
    await validateHcl(hcl);
  });

  it("serializes an output with reference value", async () => {
    const block = createBlock("output", "output", "web_ip", {});
    block.provider = "";
    block.attributes.set("value", {
      key: "value", value: "aws_instance.web.public_ip", valueType: "reference",
    });
    addBlock(config, block);

    const hcl = serializeToHcl(config);
    expect(hcl).toContain('output "web_ip"');
    expect(hcl).toContain("value = aws_instance.web.public_ip");
    await validateHcl(hcl);
  });

  it("serializes an output with string value (quoted)", async () => {
    const block = createBlock("output", "output", "vpc_desc", {});
    block.provider = "";
    block.attributes.set("value", {
      key: "value", value: "VPC for production", valueType: "string",
    });
    addBlock(config, block);

    const hcl = serializeToHcl(config);
    expect(hcl).toContain('output "vpc_desc"');
    expect(hcl).toContain('value = "VPC for production"');
    await validateHcl(hcl);
  });

  it("serializes depends_on from connections", async () => {
    const b1 = createBlock("resource", "aws_instance", "web", { ami: "ami-123" });
    const b2 = createBlock("resource", "aws_s3_bucket", "assets", { acl: "private" });
    addBlock(config, b1);
    addBlock(config, b2);

    const conn: Connection = {
      id: generateId(), sourceId: b1.id, targetId: b2.id,
      sourceLabel: "web", targetLabel: "assets",
    };
    addConnection(config, conn);

    const hcl = serializeToHcl(config);
    expect(hcl).toContain("depends_on = [aws_s3_bucket.assets]");
    await validateHcl(hcl);
  });

  it("serializes list values with proper quoting", async () => {
    const block = createBlock("resource", "aws_security_group_rule", "allow_http", {
      cidr_blocks: "[0.0.0.0/0]",
    });
    addBlock(config, block);

    const hcl = serializeToHcl(config);
    expect(hcl).toContain('cidr_blocks = ["0.0.0.0/0"]');
    await validateHcl(hcl);
  });

  it("serializes list values with references unquoted", async () => {
    const block = createBlock("resource", "aws_instance", "web", {
      security_groups: "[aws_security_group.sg.id]",
    });
    addBlock(config, block);

    const hcl = serializeToHcl(config);
    expect(hcl).toContain("security_groups = [aws_security_group.sg.id]");
    await validateHcl(hcl);
  });

  it("serializes JSON values with jsonencode", async () => {
    const block = createBlock("resource", "aws_iam_role", "role", {});
    block.attributes.set("assume_role_policy", {
      key: "assume_role_policy",
      value: '{"Version":"2012-10-17","Statement":[]}',
      valueType: "map",
    });
    addBlock(config, block);

    const hcl = serializeToHcl(config);
    expect(hcl).toContain("assume_role_policy = jsonencode(");
    await validateHcl(hcl);
  });

  it("serializes mixed list values correctly", async () => {
    const block = createBlock("resource", "aws_instance", "web", {
      values: "[80, true, custom_value]",
    });
    addBlock(config, block);

    const hcl = serializeToHcl(config);
    expect(hcl).toContain('values = [80, true, "custom_value"]');
    await validateHcl(hcl);
  });

  it("serializes expression valueType unquoted", async () => {
    const block = createBlock("resource", "aws_iam_role", "role", {});
    block.attributes.set("assume_role_policy", {
      key: "assume_role_policy",
      value: 'jsonencode({Version: "2012-10-17", Statement: [{Effect: "Allow", Principal: {Service: "ec2.amazonaws.com"}, Action: "sts:AssumeRole"}]})',
      valueType: "expression",
    });
    addBlock(config, block);

    const hcl = serializeToHcl(config);
    // Expression type renders without wrapping quotes — it's a raw HCL function call
    expect(hcl).toContain("assume_role_policy = jsonencode(");
    expect(hcl).not.toContain('"jsonencode(');
    await validateHcl(hcl);
  });

  it("serializes s: prefix forced strings in HCL with quotes", async () => {
    const block = createBlock("resource", "aws_db_instance", "db", {});
    // Simulating what makeAttribute produces with s: prefix
    block.attributes.set("engine_version", {
      key: "engine_version",
      value: "15",
      valueType: "string",
    });
    addBlock(config, block);

    const hcl = serializeToHcl(config);
    expect(hcl).toContain('engine_version = "15"');
    await validateHcl(hcl);
  });

  it("orders blocks: provider → variable → resource → output", async () => {
    addBlock(config, createBlock("output", "output", "ip", {}));
    addBlock(config, createBlock("resource", "aws_instance", "web", {}));
    const prov = createBlock("provider", "aws", "aws", {});
    prov.provider = "aws";
    addBlock(config, prov);
    const v = createBlock("variable", "variable", "env", {});
    v.provider = "";
    addBlock(config, v);

    const hcl = serializeToHcl(config);
    const provIdx = hcl.indexOf('provider "aws"');
    const varIdx = hcl.indexOf('variable "env"');
    const resIdx = hcl.indexOf('resource "aws_instance"');
    const outIdx = hcl.indexOf('output "ip"');
    expect(provIdx).toBeLessThan(varIdx);
    expect(varIdx).toBeLessThan(resIdx);
    expect(resIdx).toBeLessThan(outIdx);
    await validateHcl(hcl);
  });
});
