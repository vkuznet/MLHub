# MLHub inference APIs
For command line usage, first you need to obtain a token via
`/login` web API and then proceed with one of the following APIs:
- `/model/<model_name>/predict` to get prediction from a given ML model.
```
# provide prediction for given input vector
curl -X GET \
     -H "content-type: application/json" \
     -H "Authorization: Bearer $token" \
     -d '{"input": [input values]}' \
     http://localhost:port/model/mnist/predict

# provide prediction for given image file
curl http://localhost:8083/model/mnist \
     -H "Authorization: Bearer $token" \
     -F 'image=@./img4.png'
```
