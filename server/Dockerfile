FROM python:3.9.7-slim
COPY server /
# proof of mount working.
RUN rm -f /config.ini
RUN python -m unittest tests/test_common.py
ENTRYPOINT ["/bin/sh"]
