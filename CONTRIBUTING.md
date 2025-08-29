# Contributing to NetCrate

Thank you for your interest in contributing to NetCrate! This document provides guidelines and information for contributors.

## Code of Conduct

This project adheres to a code of conduct that ensures a respectful and inclusive environment. By participating, you agree to uphold these standards.

## Legal Requirements

**⚠️ IMPORTANT: By contributing to NetCrate, you confirm that your contributions are intended solely for defensive security purposes and comply with all applicable laws.**

### Contribution Guidelines
- All code must be for defensive security testing only
- No malicious functionality will be accepted
- Contributors must ensure their code doesn't violate any laws
- All contributions are subject to legal review

## Getting Started

### Prerequisites
- Go 1.21 or higher
- Git
- Basic understanding of network security concepts

### Development Setup
```bash
# Clone the repository
git clone https://github.com/[organization]/netcrate.git
cd netcrate

# Install dependencies
go mod download

# Run tests
go test ./...

# Build the project
go build ./cmd/netcrate
```

## Contribution Process

### 1. Before You Start
- Check existing issues for similar work
- Open an issue to discuss major changes
- Ensure your contribution aligns with project goals

### 2. Development Workflow
1. Fork the repository
2. Create a feature branch: `git checkout -b feature/your-feature-name`
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass: `go test ./...`
6. Run linting: `go fmt ./...` and `go vet ./...`
7. Commit your changes with clear messages
8. Push to your fork: `git push origin feature/your-feature-name`
9. Open a Pull Request

### 3. Pull Request Guidelines
- Provide a clear description of the changes
- Reference related issues
- Include tests for new features
- Ensure documentation is updated
- Follow the existing code style

## Code Standards

### Go Style Guide
- Follow standard Go formatting (`go fmt`)
- Use meaningful variable and function names
- Add comments for exported functions and types
- Keep functions small and focused
- Handle errors appropriately

### Security Considerations
- All network operations must respect rate limits
- Validate all user inputs
- Never log sensitive information
- Follow the principle of least privilege
- Implement proper error handling

### Testing Requirements
- Unit tests for all new functions
- Integration tests for new features
- Test both success and failure cases
- Maintain test coverage above 80%

## Documentation

### Code Documentation
- Document all exported functions and types
- Include usage examples in comments
- Keep documentation up to date with code changes

### User Documentation
- Update README.md for new features
- Add or update man pages for new commands
- Include examples in documentation

## Issue Reporting

### Bug Reports
Please include:
- Clear description of the issue
- Steps to reproduce
- Expected vs actual behavior
- Environment details (OS, Go version, etc.)
- Log output (if applicable)

### Feature Requests
Please include:
- Clear description of the proposed feature
- Use case and justification
- Proposed implementation approach
- Potential security implications

## Security Vulnerabilities

**Do not report security vulnerabilities through public issues.**

Instead, please:
1. Email security@[domain] with details
2. Include steps to reproduce
3. Allow time for assessment and patching
4. Follow responsible disclosure practices

## Community

### Communication Channels
- GitHub Issues for bug reports and feature requests
- GitHub Discussions for general questions
- Pull Requests for code contributions

### Getting Help
- Check existing documentation and issues first
- Provide clear, specific questions
- Include relevant context and environment details

## Recognition

Contributors will be recognized in:
- CONTRIBUTORS.md file
- Release notes for significant contributions
- Project documentation where appropriate

## License

By contributing to NetCrate, you agree that your contributions will be licensed under the MIT License.

---

**Thank you for helping make NetCrate better!**

*Last updated: 2025-08-28*