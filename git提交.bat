@echo off
echo.ÊäÈë°æ±¾Ãû³Æ
set /p name=
git add .
git commit -m "%name%"
git push origin
pause