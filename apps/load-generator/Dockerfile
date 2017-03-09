FROM python:3

COPY main.py /usr/src/app/
COPY requirements.txt /usr/src/app/

RUN pip install -r /usr/src/app/requirements.txt

WORKDIR /usr/src/app

ENTRYPOINT ["python", "-u", "./main.py"]