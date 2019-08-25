FROM python:3

COPY tools/load-generator/requirements.txt /usr/src/app/
RUN pip install -r /usr/src/app/requirements.txt

COPY tools/load-generator/main.py /usr/src/app/

WORKDIR /usr/src/app

ENTRYPOINT ["python", "-u", "./main.py"]
