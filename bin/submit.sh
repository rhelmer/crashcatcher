curl -H "Host: crash-reports.example.com" -F "ProductName=WaterWolf" -F "Version=1.2.3" -F "Otherdata=Something" -F "upload_file_minidump=@testdata/breakpad_test.dump" http://localhost:8080/submit
echo
