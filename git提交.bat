@echo off
echo.����汾����
set /p name=
git add .
git commit -m "%name%"
git push origin
git push github
pause