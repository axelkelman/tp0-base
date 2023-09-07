from .utils import Bet


BET_PKT = 1
BET_ACK_PKT = 2

class PacketHeader:
    """Header common to every packet in the protocol"""
    def __init__(self,pkt_type,id):
        self.packet_type = pkt_type
        self.id = id
        
    def header_to_bytes(self,payload_size):
        return [self.packet_type,self.id,payload_size + 3]
        

class BetPacket:
    """A packet containing information of a bet"""
    def __init__(self,bet):
        self.pkt_type = BET_PKT
        self.bet = bet
        
def bet_from_bytes(bytes):
    """Converts an array of bytes into a BetPacket"""
    msg = ''.join(chr(i) for i in bytes[3:])
    agency = str(bytes[1])
    fields = msg.split("|")
    bet = Bet(agency,fields[0],fields[1],fields[2],fields[3],fields[4])
    return BetPacket(bet)


class BetAckPacket:
    """Bet Ack packet sent to the client"""
    def __init__(self,document,number,status,id):
        self.header = PacketHeader(BET_ACK_PKT,id)
        self.document = document
        self.number = number
        self.status = status
        
    def bet_ack_to_bytes(self):
        """Converts a BetAckPacket to an array of bytes"""
        format_payload = self.document + "|" + self.number + "|" + self.status
        header_bytes = self.header.header_to_bytes(len(format_payload))
        return bytearray(header_bytes) + format_payload.encode('utf-8')
         
