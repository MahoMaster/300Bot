timeout /t 1 /nobreak >nul
start open_gocqhttp.bat
timeout /t 20 /nobreak >nul
call "./300Bot.exe"