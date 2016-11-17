import gc
import go_vncdriver
import time

session = go_vncdriver.VNCSession()
session.start_profile("/tmp/profile.pprof")
session.connect("conn1", address="127.0.0.1:5900", encoding="tight")


for i in range(10000):
    # observations, infos, errors = session.step({"conn1": [("KeyEvent", 1, 2)]})
    # session.render("conn1")
    # print(observations, infos, errors)
    # time.sleep(0.016)

    observations, infos, errors = session.step({"conn1": [("KeyEvent", 1, 2)]})
    # gc.collect()
    # print(observations, infos, errors)
    # session.connect("conn1", address="127.0.0.1:5900", encoding="tight")

session.end_profile()

# session.close("conn1")
# for i in range(10):
#     observations, infos, errors = session.step({"conn2": [("KeyEvent", 1, 2)]})
#     print(observations, infos, errors)
#     time.sleep(0.016)

# session.render("first")
