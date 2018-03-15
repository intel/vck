FROM python:2.7.14

RUN apt-get update &&  apt-get install -y git clang libcurl4-openssl-dev libopencv-dev libsox-dev libboost-filesystem-dev libboost-system-dev libssl-dev cmake

RUN git clone https://github.com/NervanaSystems/aeon.git aeon; cd aeon && pip install -r requirements.txt && mkdir -p build && cd build && cmake .. && pip install .

COPY aeon_test.py /
COPY train.csv /
COPY images /images
