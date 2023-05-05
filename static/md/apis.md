# MLHub APIs
- `/model/<name>` end-point provides the following methods:
  - `GET` HTTP request will retrieve ML meta-data for provide ML name, e.g.
```
# fetch meta-data info about ML model
curl http://localhost:port/model/mnist
```
  - `POST` HTTP request will create new ML entry in MLHub for provided
  ML meta-data JSON record and ML tarball
```
# post ML meta-data
curl -X POST \
     -H "content-type: application/json" \
     -d '{"model": "mnist", "type": "TensorFlow", "meta": {}}' \
     http://localhost:port/model/mnist
```
  - `PUT` HTTP request will update exsiting ML entry in MLHub for provided
  ML meta-data JSON record
```
# post ML meta-data
curl -X PUT \
     -H "content-type: application/json" \
     -d '{"model": "mnist", "type": "TensorFlow", "meta": {"param": 1}}' \
     http://localhost:port/model/mnist
```
  - `DELETE` HTTP request will delete ML entry in MLHub for provided ML name
```
curl -X DELETE \
     http://localhost:port/model/mnist
```
- `/models` to list existing ML models, GET HTTP request
```
# to get all ML models
curl http://localhost:port/models
```

### ML model APIs
- `/model/<model_name>/upload` uploads ML model bundle
```
# upload ML model
curl -X POST -H "Content-Encoding: gzip" \
     -H "content-type: application/octet-stream" \
     --data-binary @./mnist.tar.gz \
     http://localhost:port/model/mnist/upload
```
- `/model/<model_name>/download` downloads ML model bundle
```
curl http://localhost:port/model/mnist/download
```
- `/model/<model_name>/predict` to get prediction from a given ML model.
```
# provide prediction for given input vector
curl -X GET \
     -H "content-type: application/json" \
     -d '{"input": [input values]}' \
     http://localhost:port/model/mnist/predict

# provide prediction for given image file
curl http://localhost:8083/model/mnist \
     -F 'image=@./img4.png'
```
