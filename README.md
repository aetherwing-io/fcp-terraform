# fcp-terraform

MCP server for Terraform HCL generation through intent-level commands.

## What It Does

fcp-terraform lets LLMs build Terraform configurations by describing infrastructure intent -- resources, data sources, variables, outputs -- and renders them into valid HCL. Instead of writing raw HCL syntax, the LLM sends operations like `add resource aws_instance web ami:"ami-0c55b159" instance_type:t2.micro` and fcp-terraform manages the semantic model, dependency graph, and serialization. Built on the [FCP](https://github.com/aetherwing-io/fcp) framework.

## Quick Example

```
terraform_session('new "Main Infrastructure"')

terraform([
  'add resource aws_instance web ami:"ami-0c55b159" instance_type:t2.micro',
  'add resource aws_s3_bucket assets bucket:"my-assets"',
  'add variable region default:"us-east-1" type:string',
  'add output instance_ip value:"aws_instance.web.public_ip"',
])

terraform_query('plan')
```

The `plan` query produces:

```hcl
variable "region" {
  type    = string
  default = "us-east-1"
}

resource "aws_instance" "web" {
  ami           = "ami-0c55b159"
  instance_type = "t2.micro"
}

resource "aws_s3_bucket" "assets" {
  bucket = "my-assets"
}

output "instance_ip" {
  value = aws_instance.web.public_ip
}
```

### Available MCP Tools

| Tool | Purpose |
|------|---------|
| `terraform(ops)` | Batch mutations -- add, set, remove, connect, nest, label, style |
| `terraform_query(q)` | Inspect the config -- map, list, describe, plan, graph, validate, find |
| `terraform_session(action)` | Lifecycle -- new, open, save, checkpoint, undo, redo |
| `terraform_help()` | Full reference card |

### Supported Block Types

| Verb | Syntax |
|------|--------|
| `add resource` | `add resource TYPE LABEL [key:value...]` |
| `add provider` | `add provider PROVIDER [region:R] [key:value...]` |
| `add variable` | `add variable NAME [type:T] [default:V] [description:D]` |
| `add output` | `add output NAME value:EXPR [description:D]` |
| `add data` | `add data TYPE LABEL [key:value...]` |
| `add module` | `add module LABEL source:PATH [key:value...]` |
| `connect` | `connect SRC -> TGT [label:TEXT]` |
| `set` | `set LABEL key:value [key:value...]` |
| `nest` | `nest LABEL BLOCK_TYPE [key:value...]` |
| `remove` | `remove LABEL` or `remove @SELECTOR` |

### Selectors

```
@type:aws_instance      All resources of a given type
@provider:aws            All blocks from a given provider
@kind:resource           All blocks of kind (resource, variable, output, data)
@tag:KEY or @tag:KEY=VAL Blocks matching a tag
@all                     All blocks
```

## Installation

Requires Node >= 18.

```bash
npm install @aetherwing/fcp-terraform
```

### MCP Client Configuration

```json
{
  "mcpServers": {
    "terraform": {
      "command": "node",
      "args": ["node_modules/@aetherwing/fcp-terraform/dist/index.js"]
    }
  }
}
```

## Architecture

3-layer architecture:

```
MCP Server (Intent Layer)
  src/server/ -- Parses op strings, resolves refs, dispatches
        |
Semantic Model (Domain)
  src/model/ -- In-memory Terraform graph (resources, data, variables, outputs)
  src/types/ -- Core TypeScript interfaces
        |
Serialization (HCL)
  src/hcl.ts -- Semantic model -> HCL text output
```

Supporting modules:

- `src/ops.ts` -- Operation string parser
- `src/verbs.ts` -- Verb specs and reference card
- `src/queries.ts` -- Query dispatcher (map, plan, graph, validate, etc.)
- `src/adapter.ts` -- FCP core adapter

Provider is auto-detected from resource type prefixes (`aws_`, `google_`, `azurerm_`).

## Worked Example: AWS Production Web Stack

A realistic deployment showing how operations compose. This example creates a VPC with subnets, EC2, RDS, S3, IAM, and security groups -- 13 resources total.

```
terraform_session('new "Acme Corp Web Stack"')

# Provider + Variables
terraform([
  'add provider aws region:us-east-1',
  'add variable environment type:string default:"production" description:"Deployment environment"',
  'add variable project_name type:string default:"acme-web" description:"Project name"',
  'add variable vpc_cidr type:string default:"10.0.0.0/16" description:"VPC CIDR block"',
  'add variable instance_type type:string default:"t3.medium" description:"EC2 instance type"',
  'add variable db_instance_class type:string default:"db.t3.medium" description:"RDS instance class"',
])

# Networking
terraform([
  'add resource aws_vpc main cidr_block:var.vpc_cidr enable_dns_support:true enable_dns_hostnames:true',
  'add resource aws_subnet public_a vpc_id:aws_vpc.main.id cidr_block:"10.0.1.0/24" map_public_ip_on_launch:true',
  'add resource aws_subnet public_b vpc_id:aws_vpc.main.id cidr_block:"10.0.2.0/24" map_public_ip_on_launch:true',
  'add resource aws_internet_gateway igw vpc_id:aws_vpc.main.id',
  'add resource aws_route_table public vpc_id:aws_vpc.main.id',
])

# Nested blocks for route table and security groups
terraform([
  'nest public route cidr_block:"0.0.0.0/0" gateway_id:aws_internet_gateway.igw.id',
  'add resource aws_security_group web name:"${var.project_name}-web-sg" vpc_id:aws_vpc.main.id',
  'nest web ingress from_port:80 to_port:80 protocol:"tcp"',
  'nest web ingress from_port:443 to_port:443 protocol:"tcp"',
  'nest web egress from_port:0 to_port:0 protocol:"-1"',
])

# Compute + Database
terraform([
  'add resource aws_instance webserver ami:"ami-0c55b159" instance_type:var.instance_type subnet_id:aws_subnet.public_a.id',
  'nest webserver root_block_device volume_size:20 volume_type:"gp3"',
  'add resource aws_db_subnet_group dbsubnet name:"${var.project_name}-db-subnet"',
  'set dbsubnet subnet_ids:"[aws_subnet.public_a.id,aws_subnet.public_b.id]"',
  'add resource aws_db_instance rds engine:"postgresql" instance_class:var.db_instance_class allocated_storage:20',
  'add resource aws_s3_bucket assets bucket:"${var.project_name}-assets-${var.environment}"',
])

# Outputs
terraform([
  'add output vpc_id value:aws_vpc.main.id',
  'add output web_ip value:aws_instance.webserver.public_ip',
  'add output db_endpoint value:aws_db_instance.rds.endpoint',
])

# Tags (all resources at once)
terraform([
  'style main tags:"Name=${var.project_name}-vpc,Environment=${var.environment}"',
  'style public_a tags:"Name=${var.project_name}-public-a,Environment=${var.environment}"',
  'style public_b tags:"Name=${var.project_name}-public-b,Environment=${var.environment}"',
  'style igw tags:"Name=${var.project_name}-igw,Environment=${var.environment}"',
  'style webserver tags:"Name=${var.project_name}-web,Environment=${var.environment}"',
  'style rds tags:"Name=${var.project_name}-db,Environment=${var.environment}"',
  'style assets tags:"Name=${var.project_name}-assets,Environment=${var.environment}"',
])

# Day-2 modifications
terraform([
  'label webserver app_server',                         # Rename
  'set instance_type default:"t3.large"',               # Change default
  'remove assets',                                       # Replace S3 bucket
  'add resource aws_s3_bucket static_assets bucket:"${var.project_name}-static-${var.environment}"',
])

terraform_query('plan')  # Export final HCL
```

## Capability Matrix

| Capability | Status | Notes |
|------------|--------|-------|
| Resources, data sources, modules | Supported | All Terraform block types |
| Variables with types and defaults | Supported | string, number, bool, list, map, object |
| Outputs | Supported | Simple value expressions |
| Nested blocks (ingress, egress, route, filter) | Supported | Via `nest` verb |
| Tags | Supported | Via `style` verb with `tags:"K=V,K2=V2"` |
| Rename blocks | Partial | `label` renames but does not cascade references |
| Selectors for bulk operations | Partial | `remove @selector` works; `style @selector` not yet supported |
| `jsonencode()` / HCL functions | Not supported | Complex expressions containing colons conflict with the DSL tokenizer |
| Nested block removal/editing | Not supported | Nested blocks are append-only |
| `count` / `for_each` meta-arguments | Not supported | `set` cannot modify meta-arguments |

## Known Limitations

The following issues were identified through competitive validation against raw HCL writing. They are tracked for resolution.

**HCL Serialization**

- **List values in nested blocks render without quotes.** Attributes like `cidr_blocks`, `security_groups`, and `values` in filter/ingress/egress blocks output `[0.0.0.0/0]` instead of `["0.0.0.0/0"]`. This produces invalid HCL. Workaround: post-process the output or use `terraform_query('plan')` output as a starting point for manual fixes.

- **String values may lose their type.** Setting `engine_version:"15"` renders as `engine_version = 15` (number). The serializer does not preserve explicit string quoting for numeric-looking values.

- **Output values with interpolation render unquoted.** An output `value:"VPC: ${aws_vpc.main.id}"` renders without surrounding quotes.

**Labels**

- **Labels are globally unique.** You cannot use `main` for both `aws_vpc.main` and `aws_internet_gateway.main` even though Terraform allows this (labels are scoped per type). Use distinct labels like `igw`, `rds`, `dbsubnet`.

- **`label` rename breaks the lookup index.** After renaming a block with `label old new`, subsequent `style new` or `set new` calls may return "block not found". The internal index is not rebuilt after rename.

**Nested Blocks**

- **Cannot remove or edit specific nested blocks.** Once added via `nest`, a nested block (ingress rule, route, filter) cannot be individually removed or modified. If a `nest` call produces an error but partially succeeds, the ghost block persists.

**Expressions**

- **JSON with colons conflicts with the DSL tokenizer.** IAM `assume_role_policy` or similar JSON-containing attributes are parsed as key:value pairs by the FCP tokenizer, producing bare JSON objects instead of quoted strings or `jsonencode()` calls.

## Development

```bash
npm install
npm run build     # tsc
npm test          # vitest, 138 tests
```

## License

MIT

