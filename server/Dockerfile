FROM python:3.9.7-slim
COPY server /
# proof of mount working.
RUN rm -f /config.ini
# proof of mount working.
RUN python -m unittest tests/test_common.py
ENTRYPOINT ["/bin/sh"]
