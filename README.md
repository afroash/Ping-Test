## PING TEST TOOL

This tool allows you to test network connectivity to various servers by performing ping tests. It uses SSH to connect to the servers and execute the ping commands.

### Features

- Connects to multhostle servers via SSH
- Executes ping tests to a specified target host
- Displays results in a table format
- Logs results to a file (`network_tests.log`)

### Usage

1. Ensure you have the necessary dependencies installed. You can install them using:

    ```sh
    go mod tidy
    ```

2. Update the  file with the details of the servers you want to test. Example:

    ```yaml
    servers:
      - name: "Server1"
        host: "192.168.121.101"
        user: "admin"
        password: "admin"
      - name: "Server2"
        host: "192.168.121.102"
        user: "admin"
        password: "admin"
      - name: "Server3"
        host: "192.168.121.104"
        user: "root"
        password: "admin"
      - name: "Server4"
        host: "192.168.121.105"
        user: "root"
        password: "admin"
    ```

3. Run the tool:

    ```sh
    go run main.go
    ```

4. Follow the on-screen instructions to enter the target host for the ping tests.

### Dependencies

- [github.com/charmbracelet/bubbles](https://github.com/charmbracelet/bubbles)
- [github.com/charmbracelet/bubbletea](https://github.com/charmbracelet/bubbletea)
- [github.com/charmbracelet/lhostgloss](https://github.com/charmbracelet/lhostgloss)
- [golang.org/x/crypto](https://pkg.go.dev/golang.org/x/crypto)
- [gopkg.in/yaml.v2](https://gopkg.in/yaml.v2)

### License

This project is licensed under the MIT License.

