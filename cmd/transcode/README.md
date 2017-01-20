This command reads in an FBS 1.2 file (OpenAI-specific format) of messages sent
from a server to a client in a VNC session. It transcodes FramebufferUpdate
messages from their original encoding to an equivalent message with raw
encoding.

`Dockerfile` defines a tiny docker image that contains only the `transcode`
executable. The resulting image is about 6MB.

Use the image like so:

```
docker run -v volume:mount docker.openai.com/transcode -in=server.tight.fbs -out=server.raw.fbs
```
