FROM python:2

RUN pip install requests pyyaml schedule pyopenssl

COPY main.py /usr/src/app/

WORKDIR /usr/src/app

CMD ["python", "./main.py"]