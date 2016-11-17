import gc
import go_vncdriver
import time

session = go_vncdriver.VNCSession()
session.connect("conn1", address="172.16.163.128:5900", encoding="tight", subscription=[(0, 100, 0, 100)])
# session.connect("conn2", address="172.16.163.128:5900", encoding="tight")

for i in range(10):
    session.render("conn1")
    observations, infos, errors = session.step({"conn1": [("KeyEvent", 1, 2)]})
    if errors.get("conn1"):
        print("error", errors.get("conn1"))
    time.sleep(0.2)

session.update("conn1", [(200, 100, 200, 100), (400, 100, 400, 100)])

for i in range(10):
    session.render("conn1")
    observations, infos, errors = session.step({"conn1": [("KeyEvent", 1, 2)]})
    if errors.get("conn1"):
        print("error", errors.get("conn1"))
    time.sleep(0.2)


# session.close("conn1")
# for i in range(10):
#     observations, infos, errors = session.step({"conn2": [("KeyEvent", 1, 2)]})
#     print(observations, infos, errors)
#     time.sleep(0.016)

# session.render("first")
