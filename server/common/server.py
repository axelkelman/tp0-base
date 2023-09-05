import socket
import logging

from .protocol import BATCH_PKT, batch_from_bytes, FINISHED_PKT, BatchAckPacket
from .utils import store_bets

BLOCK_SIZE = 8192

class Server:
    def __init__(self, port, listen_backlog):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self._sigterm_received = False

    def run(self):
        """
        Dummy Server loop

        Server that accept a new connections and establishes a
        communication with a client. After client with communucation
        finishes, servers starts to accept new connections again
        """

        # TODO: Modify this program to handle signal to graceful shutdown
        # the server
        while not self._sigterm_received:
            client_sock = self.__accept_new_connection()
            if client_sock is not None:
                self.__handle_client_connection(client_sock)        

    def __handle_client_connection(self, client_sock):
        """
        Reads batch from the client until the client is done

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """
        try:
            while True:     
                bytes,addr = self.__read_client_socket(client_sock)
                if bytes == 0:
                    logging.error("action: receive_message | result: fail")
                    client_sock.close()
                    return
                if bytes[0] == BATCH_PKT:
                    packet = batch_from_bytes(bytes)
                    store_bets([b.bet for b in packet.bets])
                    logging.info(f'action: receive_batch | result: success | ip: {addr[0]}')                    
                if bytes[0] == FINISHED_PKT:
                    logging.info(f'action: receive_finished_pkt | result: success | ip: {addr[0]}')
                    msg = BatchAckPacket("1",bytes[1]).bet_ack_to_bytes()
                    break                   
                
            self.__write_client_socket(msg,client_sock)
        except OSError as e:
            logging.error("action: receive_message | result: fail | error: {e}")
        finally:
            client_sock.close()

    def __accept_new_connection(self):
        """
        Accept new connections

        Function blocks until a connection to a client is made.
        Then connection created is printed and returned
        """

        # Connection arrived
        logging.info('action: accept_connections | result: in_progress')
        try:
            c, addr = self._server_socket.accept()
        except OSError:
            return
        logging.info(f'action: accept_connections | result: success | ip: {addr[0]}')
        return c 
    
    def _sigterm_handler(self,signum,frame):
        logging.info('action: receive_sigterm')
        self._sigterm_received = True
        logging.info('action: closing server socket | result: in_progress')
        self._server_socket.close()
        logging.info('action: closing server socket | result: sucess')
        
    def __read_client_socket(self,client_sock):
        """
        Reads message from a specific socket client. Reads until it reaches
        BLOCK_SIZE amount of bytes
        """
        bytes_read = 0
        bytes = []
        size_of_packet = 0
        size_read = False
        while bytes_read < BLOCK_SIZE:
            bytes += list(client_sock.recv(BLOCK_SIZE - bytes_read))
            bytes_read = len(bytes)
            if not size_read:
                if bytes_read == 0:
                    return 0,None
                size_of_packet = (bytes[3] << 8) | bytes[2]
                size_read = True                
        
        addr = client_sock.getpeername()
        return bytes[:size_of_packet],addr
    
    def __write_client_socket(self,msg,client_sock):
        """
        Writes message to a specific socket client. Adds necessary padding to reach
        BLOCK_SIZE amount of bytes to send
        """
        sent_bytes = 0
        padding_length = BLOCK_SIZE - len(msg)
        message = msg + (b'\x00' * padding_length)
        while sent_bytes < BLOCK_SIZE:
            sent = client_sock.send(message[sent_bytes:])
            sent_bytes += sent
            
        