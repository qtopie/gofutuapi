# Project Instructions - go-futu-api

## Core Mandates
- **Go-Only Trading**: NEVER use Python scripts or the Python SDK for trading operations (placing, modifying, or cancelling orders). ALL trading must be performed using the local Go implementation (`gofutuapi`).
- **Tactical Rounding Preference**: When setting limit orders, prefer using small increments/decimals (e.g., $395.20 instead of $395.00) to improve the probability of prior fulfillment at psychological support levels.
- **Context Preservation**: Strategic market analysis and investment decisions should be recorded in the private memory folder (`~/.gemini/tmp/go-futu-api/memory/`).
- **FutuOpenD GUI Startup**: Always start FutuOpenD GUI using `copilot-infra` as a background task.
  - If `copilot-infra` is not running, start it
  - Submit the GUI task: 
    ```bash
    curl -X POST http://localhost:18080/tasks \
         -H "Content-Type: application/json" \
         -d '{
           "name": "futu-gui",
           "cmd": "/home/qtopierw/workspace/projects/go-futu-api/.opend/app/Futu_OpenD_10.2.6208_Ubuntu18.04/Futu_OpenD-GUI_10.2.6208_Ubuntu18.04/Futu_OpenD-GUI_10.2.6208_Ubuntu18.04.AppImage --no-sandbox"
         }'
    ```

## Implementation Details
- Use the `gofutuapi` library for all interactions with OpenD.
- **Complete Portfolio Discovery**: When querying for accounts using `TRD_GETACCLIST`, ALWAYS set the `NeedGeneralSecAccount` flag to `true` in the `C2S` request. Failing to do so will hide Universal Accounts (全能账户) and their associated holdings.
- **Account-Specific Positions**: Use `GetPositionsForAccount` to retrieve holdings. Ensure you iterate through all authorized markets listed in `acc.TrdMarketAuthList` to get a complete view of multi-market accounts.
- When creating new tools in the `cmd/` directory, follow the existing patterns seen in `cmd/kline` or `cmd/analyze_favorites`.
- Ensure all Go tools handle OpenD connections safely and provide clear output for the user.
