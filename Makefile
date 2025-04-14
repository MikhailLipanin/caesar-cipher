build-dll:
	go build -buildmode=plugin -o .build/caesar_keyword.so caesar_keyword.go
