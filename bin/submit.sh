curl -H "Host: crash-reports.example.com" -F "ProductName=Nagios-Check" -F "Version=3.5.8" -F "upload_file_minidump=@testdata/breakpad_test.dump" http://localhost:8080/submit
echo
