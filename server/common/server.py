import socket
import logging
import multiprocessing

from .protocol import BATCH_PKT, batch_from_bytes, FINISHED_PKT, BatchAckPacket, WINNER_PKT, WinnerPacket
from .utils import store_bets, load_bets, has_won

BLOCK_SIZE = 8192
CLIENTS_AMOUNT = 5
WINNER_NOT_READY_STATUS = "0"
WINNER_READY_STATUS = "1"

class Server:
    def __init__(self, port, listen_backlog):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self._sigterm_received = False
        self._main_sigterm_received = multiprocessing.Value('i',0)
        self._main_sigterm_received_lock = multiprocessing.Lock()
        

    def run(self):
        """
        Dummy Server loop

        Server that accept a new connections and establishes a
        communication with a client. After client with communucation
        finishes, servers starts to accept new connections again
        """

        clients_ready = multiprocessing.Array('i',CLIENTS_AMOUNT)
        clients_ready_lock = multiprocessing.Lock()
        utils_function_lock = multiprocessing.Lock()
        sigterm_received = self._main_sigterm_received
        sigterm_received_lock = self._main_sigterm_received_lock
        process_handlers = []
        i = 0
        while not self._sigterm_received and i < CLIENTS_AMOUNT:
            client_sock = self.__accept_new_connection()
            if client_sock is not None:
                args = (client_sock,clients_ready,clients_ready_lock,utils_function_lock,sigterm_received,sigterm_received_lock,) 
                p = multiprocessing.Process(target=self.__handle_client_connection,args = args)
                p.start()
                process_handlers.append(p)
                i += 1
                
        for p in process_handlers:
            p.join()
            
        logging.info("action: finishing_server")
             

    def __handle_client_connection(self, client_sock,clients_ready,clients_ready_lock,utils_function_lock,sigterm_received,sigterm_received_lock):
        """
        Reads messages from the client until the client gets the lottery results

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """
        try:
            while True:     
                bytes,addr = self.__read_client_socket(client_sock)
                sigterm_received_lock.acquire()
                sigterm = sigterm_received.value
                sigterm_received_lock.release()
                if sigterm:
                    logging.info("action: sigterm_received_child")
                    break                    
                if bytes == 0:
                    logging.error("action: receive_message | result: fail")
                    break
                if bytes[0] == WINNER_PKT:
                    msg,status = self.__winner_handler(clients_ready,clients_ready_lock,utils_function_lock,bytes)
                    logging.info(f'action: receive_winner | result: success | ip: {addr[0]} | client: {bytes[1]} | status: {status}')
                    self.__write_client_socket(msg,client_sock) 
                    if status == WINNER_READY_STATUS:
                        break
                if bytes[0] == BATCH_PKT:
                    self.__batch_handler(utils_function_lock,bytes)
                    logging.info(f'action: receive_batch | result: success | ip: {addr[0]} | client: {bytes[1]}')                    
                if bytes[0] == FINISHED_PKT:
                    logging.info(f'action: receive_finished_pkt | result: success | ip: {addr[0]} | client: {bytes[1]}')
                    msg = self.__finished_handler(clients_ready,clients_ready_lock,bytes)
                    self.__write_client_socket(msg,client_sock)          
            
        except OSError as e:
            logging.error("action: receive_message | result: fail | error: {e}")
        finally:
            logging.info('action: closing client socket | result: in_progress')
            client_sock.shutdown(socket.SHUT_RDWR)
            client_sock.close()
            logging.info('action: closing client socket | result: sucess')

    def __get_winners(self,agency,utils_function_lock):
        """Gets the lottery winners from a certain agency"""
        utils_function_lock.acquire()
        bets = list(load_bets())
        utils_function_lock.release()
        winners = [b for b in bets if has_won(b) and b.agency == agency]
        return winners
    
    def __winner_handler(self,clients_ready,clients_ready_lock,utils_function_lock,bytes):
        """Handles the event of a winner packet sent by the client"""
        winners = []
        status = WINNER_NOT_READY_STATUS
        clients_ready_lock.acquire()
        if 0 not in clients_ready:
            winners = self.__get_winners(bytes[1],utils_function_lock)
            status = WINNER_READY_STATUS
        clients_ready_lock.release()
        msg = WinnerPacket(status,bytes[1],winners).winner_to_bytes()
        return msg,status
        
    def __batch_handler(self,utils_function_lock,bytes):
        """Handles the event of a batch packet sent by the client"""
        packet = batch_from_bytes(bytes)
        utils_function_lock.acquire()
        store_bets([b.bet for b in packet.bets])
        utils_function_lock.release()
        
    def __finished_handler(self,clients_ready,clients_ready_lock,bytes):
        """Handles the event of a finished packet sent by the client"""  
        msg = BatchAckPacket("1",bytes[1]).bet_ack_to_bytes()
        clients_ready_lock.acquire()
        clients_ready[int(bytes[1]) - 1] = 1
        clients_ready_lock.release()
        return msg
    
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
        self._server_socket.shutdown(socket.SHUT_RDWR)
        self._server_socket.close()
        logging.info('action: closing server socket | result: sucess')
        self._main_sigterm_received_lock.acquire()
        self._main_sigterm_received.value = 1
        self._main_sigterm_received_lock.release()
        
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
            
        