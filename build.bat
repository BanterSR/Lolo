@echo off
setlocal enabledelayedexpansion

:start
cls
echo ***************************************
echo *       Lolo Build System Menu        *
echo ***************************************
echo.
echo 1. Build Lolo
echo 2. Exit
echo.
set /p choice="Please enter your choice (1-2): "

if "%choice%"=="1" goto build_Lolo
if "%choice%"=="2" exit /b

echo Invalid choice, please try again.
pause
goto start

:build_Lolo
set "OUT_DIR=./bin/Lolo"
set "MAIN_PATH=.\main.go"
set "OUTPUT_NAME=Lolo"
set "BUILD_TAGS="
set "TAG_SUFFIX="

:: Tag selection for Lolo
cls
echo ***************************************
echo *        Lolo Build Tags Menu         *
echo ***************************************
echo.
set /p BUILD_TAGS="Enter build tags (comma separated, leave empty for none): "

:: Ask if user wants to add 'lite' tag
set "ADD_LITE="
set /p ADD_LITE="Add 'lite' tag? (y/n, default=n): "
if /i "!ADD_LITE!"=="y" (
    if defined BUILD_TAGS (
        set "BUILD_TAGS=!BUILD_TAGS!,lite"
    ) else (
        set "BUILD_TAGS=lite"
    )
)

:: Create tag suffix for filename
if defined BUILD_TAGS (
    set "TAG_SUFFIX=!BUILD_TAGS:,=_!"
    set "TAG_SUFFIX=!TAG_SUFFIX: =!"
)
call :build
goto :eof

:build_all
call :build_Lolo
goto end



:build
cls
echo Building !OUTPUT_NAME! for all platforms...
if defined BUILD_TAGS (
    echo Using build tags: !BUILD_TAGS!
    echo Tag suffix for filename: !TAG_SUFFIX!
)
echo.

:: Ensure output directory exists (recursively)
if not exist "!OUT_DIR!\" (
    echo Creating output directory: !OUT_DIR!
    mkdir "!OUT_DIR!" || (
        echo Error: Failed to create directory: !OUT_DIR!
        pause
        goto :eof
    )
)

go mod download
go mod verify
set CGO_ENABLED=0
set "PLATFORMS=darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 linux/ppc64le linux/riscv64 linux/s390x windows/amd64 windows/arm64"

for %%p in (%PLATFORMS%) do (
    for /f "tokens=1,2 delims=/" %%a in ("%%p") do (
        set "GOOS=%%a"
        set "GOARCH=%%b"
        set "GOARM="
        set "ARCH_SUFFIX=!GOARCH!"

        echo Compiling !OUTPUT_NAME! for GOOS=!GOOS! ARCH_SUFFIX=!ARCH_SUFFIX!...

        :: Prepare output filename
        set "FILE_PREFIX=!OUTPUT_NAME!"
        if defined TAG_SUFFIX set "FILE_PREFIX=!FILE_PREFIX!_!TAG_SUFFIX!"
        if "!GOOS!"=="windows" (
            set "OUTPUT_FILE=!FILE_PREFIX!_!GOOS!_!ARCH_SUFFIX!.exe"
        ) else (
            set "OUTPUT_FILE=!FILE_PREFIX!_!GOOS!_!ARCH_SUFFIX!"
        )

        :: Prepare build command
        set "BUILD_CMD=go build -ldflags="-s -w""
        if defined BUILD_TAGS set "BUILD_CMD=!BUILD_CMD! -tags "!BUILD_TAGS!""
        
        :: Execute build
        set "GOOS=!GOOS!"
        set "GOARCH=!GOARCH!"
        if defined GOARM set "GOARM=!GOARM!"
        
        !BUILD_CMD! -o "!OUT_DIR!/!OUTPUT_FILE!" %MAIN_PATH%

        :: Check if the file was created
        if not exist "!OUT_DIR!/!OUTPUT_FILE!" (
            echo Error: Failed to build !OUTPUT_FILE! for !GOOS!/!GOARCH!
        ) else (
            echo Success: Built !OUTPUT_FILE! for !GOOS!/!GOARCH!
        )
    )
)
goto :eof

:end
echo.
echo Build process completed for all projects.
echo.
pause
goto start

endlocal