FROM python:3.7

WORKDIR /app

ADD requirements.txt /app/requirements.txt
RUN pip install -r requirements.txt
COPY wait-for-app.sh /wait-for-app.sh
RUN chmod +x /wait-for-app.sh
ADD . /app

EXPOSE 80

CMD ["python", "-m", "pytest", "py.test", "-vv"]