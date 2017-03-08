## api-tester
go version http heavy-loads tester

## Download
    go get -u -v github.com/songtianyi/api-tester

## golang.org/x dep install
	mkdir $GOPATH/src/golang.org/x
	cd $GOPATH/src/golang.org/x
	git clone https://github.com/golang/net.git

## Usage
	#get help
	./api-tester
	#test url
	./api-tester -p test2.jpg -uri "http://xxx:8080/?Action=xxx&ImageName=xxx" --c 450 --n 20000

