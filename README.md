<svg height="24" width="117" xmlns="http://www.w3.org/2000/svg">
    <rect width="29" height="24" fill="#000000" />
    <rect x="29" width="88" height="24" fill="#bb400c" />
    <text text-anchor="middle" font-weight="bold" font-size="15" font-family="Verdana,Geneva,DejaVu Sans,sans-serif" fill="#ffffff" x="15" y="50%" dy=".35em">⚙️</text>
    <text text-anchor="middle" font-size="19" font-family="Verdana,Geneva,DejaVu Sans,sans-serif" fill="#ffffff" x="73" y="50%" dy=".35em">proCLI</text>
</svg>

**ProCLI** is a command-line tool designed to help developers manage and validate project prerequisites. It simplifies the setup and ensures consistency by checking for required tools, environment variables, tokens, and version control systems.

---

## Features

- **Initialize Project Configurations**:
  - Use the `init` command to interactively create a project configuration file.
  - Supports specifying:
    - Required tools
    - Environment variables
    - Tokens
    - Version control systems

- **Validate Project Setup**:
  - Use the `check` command to validate if the system meets the project prerequisites.
  - Provides a clear, actionable output with success and failure indicators.

- **Configuration Management**:
  - Configuration files are stored locally in `~/.config/procli/config.yaml`.
  - Supports multiple projects and a default project.

---

## Installation

### Prerequisites
- [Go](https://golang.org/dl/) 1.20 or later installed.

### Clone and Build
1. Clone the repository:
   ```bash
   git clone <repository-url>
   cd procli
   ```
2. Build the binary:
   ```bash
   go build -o procli
   ```

3. Install `procli` 
   ```bash
   go install 
   ```

---

## Usage

### Initialize a Project
Run the `init` command to create a new project configuration:
```bash
./procli init
```
Example interaction:
```plaintext
Enter project name: tensorflow
Enter required tools (comma-separated): docker, python
Enter environment variables (comma-separated): INDIVIDUAL_CLA, CORPORATE_CLA
Enter required tokens (comma-separated): 
Enter version control system (e.g., git): git
Project configuration saved!
```

### Validate a Project
Run the `check` command to validate project prerequisites:
```bash
./procli check <project-name>
```
If a default project is configured, the project name can be omitted:
```bash
./procli check
```

Example output:
```plaintext
Checking prerequisites for project: tensorflow

Required Tools:
✅ docker
❌ python: Tool not found in PATH

Environment Variables:
✅ INDIVIDUAL_CLA
❌ CORPORATE_CLA: Variable not set

Version Control:
✅ git

Check complete!
```

### List Configurations
To view the current configurations and their file location:
```bash
./procli list
```

Example output:

```plaintext
Configuration file location:
/home/user/.config/procli/config.yaml

Default Project:
tensorflow

Projects:
- tensorflow
  - Tools: docker, python
  - Environment Variables: INDIVIDUAL_CLA, CORPORATE_CLA
  - Version Control: git
```

---

## Configuration File Structure

Configurations are stored as YAML in `~/.config/procli/config.yaml`. Example structure:
```yaml
default: tensorflow
projects:
  tensorflow:
    required_tools:
      - docker
      - python
    environment_vars:
      - INDIVIDUAL_CLA
      - CORPORATE_CLA
    required_tokens: []
    version_control: git
```

---

## Contributing

Contributions are welcome! Please follow these steps:
1. Fork the repository.
2. Create a feature branch (`git checkout -b feature-name`).
3. Commit your changes (`git commit -m "Add feature"`).
4. Push to the branch (`git push origin feature-name`).
5. Open a Pull Request.

---

## License

This project is licensed under the MIT License. See the [LICENSE](./LICENSE) file for details.

---

## Roadmap

- Integrate a TUI (using Bubble Tea) for project initialization and editing.
