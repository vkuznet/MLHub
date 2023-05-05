# MLHub inference APIs
For command line usage, please use one of the following APIs:

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
