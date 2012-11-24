Very minimal PoC breakpad crash collector/processor in Go.

1. Build breakpad 
```
  ./bin/build_breakpad.sh
```

2. Run tests and build crashcatcher
```
  go test
  go build
```

3. Run crashcatcher (data is stored in ```./crashdata```)
```
  ./crashcatcher
```

4. Submit test crashes (test data is stored in ```./testdata```)
```
  ./bin/submit.sh 
```
