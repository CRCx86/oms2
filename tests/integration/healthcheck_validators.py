def response_should_be_ok(response):
    assert response.content.decode("utf-8") == "ok"
