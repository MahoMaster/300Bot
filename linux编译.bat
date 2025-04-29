set GOARCH=amd64
set GOOS=linux
go build main.go
del build
rename main build
pause