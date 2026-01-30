# UIA - Windows UI Automation Bridge

This package provides a bridge between Go and Python's `pywinauto` library for Windows UI Automation support.

## Overview

Windows UI Automation (UIA) is a platform-specific API that requires native Windows libraries. Since Go doesn't have direct bindings for UIA, we use a Python subprocess to leverage `pywinauto`.

## Architecture

```
Go Worker
    │
    ├── pkg/uia/bridge.go        # Go interface
    │       │
    │       ▼
    │   Python subprocess
    │       │
    │       ▼
    │   pywinauto (via uia_helper.py)
    │       │
    │       ▼
    │   Windows UI Automation API
```

## Requirements

- Windows 10/11
- Python 3.8+ with `pywinauto` installed
- The `uia_helper.py` script in the worker's directory or in PATH

## Usage

```go
import "github.com/zoeyai/zoeyworker/pkg/uia"

// Get UI elements
elements, err := uia.GetElements(windowHandle, &uia.GetElementsOptions{
    AutomationID: "myButtonId",
    MaxDepth:     3,
})

// Click an element by AutomationID
err := uia.ClickElement(windowHandle, "myButtonId")
```

## Fallback Behavior

On non-Windows platforms or when Python/pywinauto is not available, the functions return appropriate errors. The caller should handle these cases and fall back to alternative methods (OCR, image matching).
