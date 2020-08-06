import paho.mqtt.client as mqtt
import socket
import logging
import threading
import time
import collections 
import random
from queue import Queue

# Our work queue
workQueue = Queue()

def readData():
    logging.info("Starting to read data")
    global workQueue

    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.connect((socket.gethostname(), 10000))
    
    while True:
        full_msg = ''

        while True:
            msg = s.recv(1)
            if len(msg) <= 0:
                break

            full_msg += msg.decode("ascii")
            if not full_msg.endswith("\r"):
                continue
            else:
                break

        if len(full_msg) > 0:
            workQueue.put(full_msg.strip())


def readQueue():
    def parseLine(line):
        parts = line.split(',')
        if len(parts) != 4:
            return False, parts
        else:
            return True, parts
 
    logging.info("Starting to listen on the queue")
    global workQueue    

    while True:        
        if workQueue.empty() == False:
            line = workQueue.get()        
            logging.info("<--- Pulled %s off the queue", line)
            
            goodLine, parts = parseLine(line)
            if goodLine:
                # Now we send the timestamp and sensor
                # number to the mqtt server

                ts = round(time.time())
                txline = str(ts) + "," + parts[1]
                logging.info("%s", txline)

                # Set up our mqtt connection
                client = mqtt.Client()
                client.connect('glue', 1883, 60) 
                client.publish('shopmontopic', txline)
                client.disconnect()

if __name__ == "__main__":
    format = "%(asctime)s: %(message)s"
    logging.basicConfig(format=format, level=logging.INFO,
                        datefmt="%H:%M:%S")

    logging.info("Creating thread to read data")        
    gdThread = threading.Thread(target=readData)
    gdThread.start()

    logging.info("Creating read queue thread")
    rqThread = threading.Thread(target=readQueue)
    rqThread.start()
