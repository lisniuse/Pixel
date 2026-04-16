# Step 1: Generate tray icon  (SVG -> PNG x4 -> ICO)
Push-Location tools/genicon
go run . -svg ../../assets/icon.svg -out ../../internal/tray/assets/icon.ico
Pop-Location

# Step 2: Generate EXE resource (ICO -> .syso for Windows taskbar / Explorer)
#   Install rsrc once:  go install github.com/akavel/rsrc@latest
rsrc -arch amd64 -ico internal/tray/assets/icon.ico -o cmd/pixel/resource.syso

# Step 3: Build binary
go build -ldflags="-H windowsgui" -o bin/pixel.exe ./cmd/pixel/

Write-Host "Done -> bin/pixel.exe"
