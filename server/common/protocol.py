from .utils import Bet

BET_PKT = 1
BET_ACK_PKT = 2
BATCH_PKT = 3
FINISHED_PKT = 4
BATCH_ACK_PKT = 5

class PacketHeader:
    """Header common to every packet in the protocol"""
    def __init__(self,pkt_type,id):
        self.packet_type = pkt_type
        self.id = id
        
    def header_to_bytes(self,payload_size):
        header_bytes = [self.packet_type,self.id,payload_size + 4]
        return bytearray(header_bytes[0:2]) + bytearray([header_bytes[2] & 0xFF, (header_bytes[2] >> 8) & 0xFF])
        

class BetPacket:
    """A packet containing information of a bet"""
    def __init__(self,bet):
        self.pkt_type = BET_PKT
        self.bet = bet
        
def bet_from_bytes(bytes):
    """Converts an array of bytes into a BetPacket"""
    msg = bytearray(bytes[4:]).decode('utf-8')
    agency = str(bytes[1])
    fields = msg.split("|")
    bet = Bet(agency,fields[0],fields[1],fields[2],fields[3],fields[4])
    return BetPacket(bet)


class BatchPacket:
    """A packet containing information of a batch of bets"""
    def __init__(self,bets):
        self.pkt_type = BATCH_PKT
        self.bets = bets
        
        
def batch_from_bytes(bytes):
    """Converts an array of bytes into a BatchPacket"""
    size = (bytes[3] << 8) | bytes[2] 
    bets = []
    i = 5
    while i < size:
        size_of_bet = (bytes[i + 3] << 8) | bytes[i + 2] 
        bet = bet_from_bytes(bytes[i:i + size_of_bet])
        bets.append(bet)
        i += size_of_bet
        
    return BatchPacket(bets)

class BatchAckPacket:
    """A packet sent to the client to acknowledge a batch"""
    def __init__(self,status,id):
        self.header = PacketHeader(BATCH_ACK_PKT,id)
        self.status = status
        
    def bet_ack_to_bytes(self):
        """Converts a BatchAckPacket to an array of bytes"""
        format_payload = self.status
        header_bytes = self.header.header_to_bytes(len(format_payload))
        payload_encode = format_payload.encode('utf-8')
        ret = header_bytes + payload_encode
        return ret 