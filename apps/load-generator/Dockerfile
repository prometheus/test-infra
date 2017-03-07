FROM python:3

RUN pip install requests pyyaml schedule pyopenssl

COPY main.py /usr/src/app/

WORKDIR /usr/src/app

ENTRYPOINT ["python", "./main.py"]