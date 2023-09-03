import socket
import logging

from .protocol import BET_PKT, BetAckPacket, bet_from_bytes
from .utils import store_bets


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
        Read message from a specific client socket and closes the socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """
        try:
            
            bytes,addr = self.__read_client_socket(client_sock)
            if bytes[0] == BET_PKT:
                packet = bet_from_bytes(bytes)
                logging.info(f'action: receive_bet | result: success | ip: {addr[0]}')
                store_bets([packet.bet])
                logging.info(f'action: apuesta_almacenada | result: success | dni: {packet.bet.document} | numero: {packet.bet.number}')
                bet_ack = BetAckPacket(packet.bet.document,str(packet.bet.number),"1",packet.bet.agency)
                msg = bet_ack.bet_ack_to_bytes()
            
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
        """Reads message from a specific socket client"""
        bytes_read = 0
        bytes = []
        size_of_packet = 1
        size_read = False
        while bytes_read < size_of_packet:
            bytes += list(client_sock.recv(1024))
            bytes_read += len(bytes)
            if not size_read:
                size_of_packet = bytes[2]
                size_read = True
        
        addr = client_sock.getpeername()
        return bytes,addr
    
    def __write_client_socket(self,msg,client_sock):
        """Writes message to a specific socket client"""
        sent_bytes = 0
        while sent_bytes < len(msg):
            sent = client_sock.send(msg[sent_bytes:])
            sent_bytes += sent
            
        