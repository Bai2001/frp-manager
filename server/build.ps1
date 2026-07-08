$env:GOOS="linux"; $env:GOARCH="amd64"; go build -o bin/agent ./cmd/agent
$env:GOOS="windows"; $env:GOARCH="amd64"; go build -o bin/agent.exe ./cmd/agent
