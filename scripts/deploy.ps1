param(
    [ValidateSet("build", "up", "down", "restart", "logs", "ps")]
    [string]$Action = "up"
)

$ErrorActionPreference = "Stop"
$ComposeFile = "docker-compose.yml"

function Run-Compose {
    param([string[]]$Args)
    docker compose -f $ComposeFile @Args
}

switch ($Action) {
    "build" {
        Run-Compose @("build", "--pull")
    }
    "up" {
        Run-Compose @("up", "-d", "--build")
    }
    "down" {
        Run-Compose @("down")
    }
    "restart" {
        Run-Compose @("down")
        Run-Compose @("up", "-d", "--build")
    }
    "logs" {
        Run-Compose @("logs", "-f", "--tail", "200")
    }
    "ps" {
        Run-Compose @("ps")
    }
}
