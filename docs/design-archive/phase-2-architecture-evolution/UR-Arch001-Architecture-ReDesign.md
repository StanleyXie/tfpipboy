User Requirement Document
Topic:Architecture redesign
Document No. UR-Arch001

According to PoC demo already implemented, we are clearer about the Usage Scenario and Key Features as a product.

## Objectives of the Product

- To be the CICD Workflow pipeline orchestrator for running Terraform modules in a Terminal Environment.
- Support to manage and orchestrate multiple terraform modules by using declarative configuration (for pipeline and workflow and module configuration).
- Automatic environment sensing and configuration alignment for seamless deployment integrated to pipeline workflow.
- Support for multiple cloud providers auth/login status tracking and Github Auth/Environment tracking.


## Target of First MVP

- Multiple terraform module management in a single repo.
- Declarative configuration format design for terraform module configuration. Which enable automation of multiple module running.
- TUI application for managing and orchestrating multiple terraform modules. I want to name it as Orchestrator.(Change commander-soldier naming to Orchestrator-Runner in concept)
- Each module will be orchestrated to run in a dedicated terminal process "Runner" with full output also be managed by "Orchestrator". Orchestrator manages full lifecycle of each "Runner", from initialization to destruction.
- Each Runner are separated by a unique identifier and can be managed independently.
- From Declarative Configuration of workflow, will support to define dependency between modules which also aligns to Runners
- <To be continued>


## Terraform Project Structure (Only for Prototyping & Testing Examples) This folder will be ignored by git.

- Sample Terraform Repository Path: .terraform-repo
- Github Workflow Path: .terraform-repo/.github
- Terraform Root Modules Path: .terraform-repo/source/root_modules
- Terraform Source Modules Path: .terraform-repo/source/src_modules


## Try to design Configuration Format for Terraform Modules in real case
- Terraform Modules: .terraform-repo/source/root_modules
- Config files: examples/azure-landing-zone/exp-alz
- Try to have all configuration related to module in modules.yaml, and orchestration configuration in pipeline.yaml
