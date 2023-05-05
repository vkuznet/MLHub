# MLHub download APIs
For command line usage, please use one of the following APIs:
- `/model/<name>` provides ML model meta-data
```
# fetch meta-data info about ML model
curl http://localhost:port/model/mnist
```
- `/model/<model_name>/download` downloads ML model bundle
```
curl http://localhost:port/model/mnist/download
```
