# ChargeX Go service(s) template

This is a template repository for creating SDK packages from OpenAPI specs.
It will generate both client and server boilerplate code.

## Included tools

- [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen#overview)
- [buf](https://buf.build/)
- [golangci-lint](https://golangci-lint.run/)
- Multi-stage, optimized Dockerfile
- GitHub Actions workflow for mock generation and OpenAPI spec validation & generation
- Workflows for testing and linting
- Workflow for docker image building and pushing
- Workflow for deploying to Cloud using terraform
- PR & issue templates

## Before publishing

To do:

- [ ] Update `README.md`
- [ ] Remove the `.gitkeep` files
- [ ] Remove the [proto example](proto/example/v1/example.proto)
- [ ] Update dependencies (if needed)
- [ ] Change/update contribution guidelines
- [ ] Update the [build workflow](.github/workflows/build-services.yaml)
- [ ] Configure deployment [workflow](.github/workflows/deploy-release.yaml)
- [ ] Update the docker image in the [Dockerfile](build/service/Dockerfile)
- [ ] Add the [docker-compose](deployments/docker) file for local development
- [ ] Update the [terraform](deployments/terraform) files for deployment
- [ ] Update the [Makefile](Makefile) with your service name and other configurations
- [ ] Update Mockery [configuration](.mockery.yaml)

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE.md) file for details.

## Contributing

We welcome contributions to this project! Please read our [contributing guidelines](CONTRIBUTING.md) for more
information on how to get started.