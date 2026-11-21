param(
    [string]$Replay = '.tmp/[th10]/replay/th10_05.rpy',
    [string]$Baseline = '',
    [string]$Out = '.tmp/wasmcheck',
    [switch]$SkipTests,
    [switch]$KeepTraces
)

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
Set-Location -LiteralPath $root
if (-not [System.IO.Path]::IsPathRooted($Out)) { $Out = Join-Path $root $Out }
$Out = [System.IO.Path]::GetFullPath($Out)
$cache = $env:GOCACHE
if (-not $cache) { $cache = Join-Path $Out 'gocache' }
foreach ($dir in @($Out, $cache, (Join-Path $Out 'gotmp'), (Join-Path $Out 'tmp'))) {
    New-Item -ItemType Directory -Path $dir -Force | Out-Null
}
$env:GOCACHE = $cache
$env:GOTMPDIR = Join-Path $Out 'gotmp'
$env:TMP = Join-Path $Out 'tmp'
$env:TEMP = $env:TMP
$env:TMPDIR = $env:GOTMPDIR
$summary = [System.Collections.Generic.List[string]]::new()
$previousOS, $previousArch = $env:GOOS, $env:GOARCH
$previousHeadless, $previousIDB = $env:TH10_WASM_HEADLESS, $env:TH10_WASM_FAKE_IDB

function Check([string]$name, [scriptblock]$body) {
    Write-Output "wasmcheck: $name"
    $timer = [System.Diagnostics.Stopwatch]::StartNew()
    & $body *> (Join-Path $Out "$name.txt")
    $code = $LASTEXITCODE
    $timer.Stop()
    $summary.Add("$name exit=$code seconds=$([math]::Round($timer.Elapsed.TotalSeconds,3))")
    $summary | Set-Content -LiteralPath (Join-Path $Out 'summary.txt') -Encoding utf8
    if ($code -ne 0) {
        Get-Content -LiteralPath (Join-Path $Out "$name.txt")
        throw "$name failed; traces are retained in $Out"
    }
}

try {
    Remove-Item Env:GOOS, Env:GOARCH -ErrorAction SilentlyContinue
    $native = Join-Path $Out 'th10sim.exe'
    $nativePerf = Join-Path $Out 'wasmperf.exe'
    $wasm = Join-Path $Out 'th10sim.wasm'
    $perf = Join-Path $Out 'wasmperf.wasm'
    $executor = Join-Path $root 'tools\wasmexec\run.cjs'
    $compare = Join-Path $root 'tools\wasmexec\compare.cjs'
    Check 'native-build' { go build -trimpath '-ldflags=-s -w' -o $native ./tools/th10sim }
    Check 'native-perf-build' { go build -trimpath '-ldflags=-s -w' -o $nativePerf ./tools/wasmperf }
    $env:GOOS = 'js'
    $env:GOARCH = 'wasm'
    Check 'sim-wasm-build' { go build -trimpath '-ldflags=-s -w' -o $wasm ./tools/th10sim }
    Check 'perf-wasm-build' { go build -trimpath '-ldflags=-s -w' -o $perf ./tools/wasmperf }
    Remove-Item Env:GOOS, Env:GOARCH
    $env:TH10_WASM_HEADLESS = '1'
    $env:TH10_WASM_FAKE_IDB = '0'
    foreach ($chapter in 0..5) {
        $a = Join-Path $Out "native_c$chapter.txt"
        $b = Join-Path $Out "node_c$chapter.txt"
        Check "native-c$chapter" { & $native -replay $Replay -chapter $chapter -frames 0 -out $a }
        if ($Baseline) { Check "base-c$chapter" { node $compare (Join-Path $Baseline "base_c$chapter.txt") $a } }
        Check "node-c$chapter" { node --stack-size=8192 $executor $wasm -replay $Replay -chapter $chapter -frames 0 -out $b }
        Check "compare-c$chapter" { node $compare $a $b }
        Check "profile-c$chapter" { node --stack-size=8192 $executor $perf -kind logic -replay $Replay -chapter $chapter -frames 0 -out (Join-Path $Out "profile_c$chapter.json") }
        Check "native-profile-c$chapter" { & $nativePerf -kind logic -replay $Replay -chapter $chapter -frames 0 -out (Join-Path $Out "native_profile_c$chapter.json") }
        $profile = (Get-Content -LiteralPath (Join-Path $Out "profile_c$chapter.json") -Raw | ConvertFrom-Json).Logic
        $nativeProfile = (Get-Content -LiteralPath (Join-Path $Out "native_profile_c$chapter.json") -Raw | ConvertFrom-Json).Logic
        foreach ($field in @('Frames','RNGState','RNGCount','AlternateRNGState','AlternateRNGCount')) {
            if ($profile.$field -ne $nativeProfile.$field) { throw "c$chapter profile differs in $field" }
        }
        $summary.Add("c$chapter main/alternate RNG state/count equal to native")
        $summary.Add("c$chapter frames=$($profile.Frames) mean-ms=$($profile.MeanMS) max-ms=$($profile.MaxMS) max-frame=$($profile.MaxFrame) GoHeap=$($profile.HeapAllocAfterLoad) linear=$($profile.LinearMemoryAfterLoad)")
        if (-not $KeepTraces) {
            $prefix = $Out + [System.IO.Path]::DirectorySeparatorChar
            foreach ($file in @($a, $b)) {
                $resolved = [System.IO.Path]::GetFullPath($file)
                if (-not $resolved.StartsWith($prefix, [System.StringComparison]::OrdinalIgnoreCase)) { throw 'unsafe trace cleanup path' }
                Remove-Item -LiteralPath $resolved
            }
        }
    }
    if (-not $SkipTests) {
        $env:GOOS = 'js'
        $env:GOARCH = 'wasm'
        $env:TH10_WASM_FAKE_IDB = '1'
        $execArg = 'node --stack-size=8192 "' + ($executor -replace '\\', '/') + '"'
        foreach ($package in @('browserstore','collision','datfs','input','pacing','replay','sht','std','trace','zun')) {
            Check "test-$package" { go test -p 1 -count=1 -exec $execArg "./$package" }
        }
    }
    $summary.Add("completed=$((Get-Date).ToString('o'))")
    $summary | Set-Content -LiteralPath (Join-Path $Out 'summary.txt') -Encoding utf8
    Get-Content -LiteralPath (Join-Path $Out 'summary.txt')
} finally {
    $env:GOOS, $env:GOARCH = $previousOS, $previousArch
    $env:TH10_WASM_HEADLESS, $env:TH10_WASM_FAKE_IDB = $previousHeadless, $previousIDB
}
