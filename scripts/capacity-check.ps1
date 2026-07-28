param(
  [int]$Sessions = 8,
  [int]$Events = 64,
  [string]$Timeout = '5s',
  [string]$MaxP95 = '500ms'
)

$ErrorActionPreference = 'Stop'

# The harness uses in-memory adapters only. Redirect stdout to retain its JSON
# report as a release artifact when running outside CI.
go run ./cmd/capacitycheck -sessions $Sessions -events $Events -timeout $Timeout -max-p95 $MaxP95
