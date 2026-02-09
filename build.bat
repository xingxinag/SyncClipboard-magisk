@echo off
REM Build script for Clipboard Whitelist Module (Windows)
REM 用于打包 Magisk 和 KernelSU 模块的构建脚本

echo =====================================
echo Clipboard Whitelist Module Builder
echo =====================================
echo.

REM 版本信息
set VERSION=1.0.0
set VERSION_CODE=10000

REM 清理旧的构建文件
echo [1/4] 清理旧的构建文件...
if exist clipboard_whitelist_magisk.zip del clipboard_whitelist_magisk.zip
if exist clipboard_whitelist_kernelsu.zip del clipboard_whitelist_kernelsu.zip

REM 构建 Magisk 版本
echo [2/4] 构建 Magisk 版本...
cd magisk
tar -acf ../clipboard_whitelist_magisk.zip *
cd ..
echo √ Magisk 版本构建完成: clipboard_whitelist_magisk.zip

REM 构建 KernelSU 版本
echo [3/4] 构建 KernelSU 版本...
cd kernelsu
tar -acf ../clipboard_whitelist_kernelsu.zip *
cd ..
echo √ KernelSU 版本构建完成: clipboard_whitelist_kernelsu.zip

REM 显示文件信息
echo [4/4] 构建完成!
echo.
echo 生成的文件:
dir clipboard_whitelist_*.zip
echo.
echo =====================================
echo 构建成功!
echo 版本: v%VERSION% (%VERSION_CODE%)
echo =====================================
pause
