'use client';

import { EventEmitter } from 'events';

export interface FileMetadata {
  id: string;
  name: string;
  size: number;
  type: string;
  chunks: number;
  checksum?: string;
}

export interface FileTransfer {
  metadata: FileMetadata;
  progress: number;
  status: 'pending' | 'transferring' | 'completed' | 'failed';
  chunks: ArrayBuffer[];
}

export class PeerConnection extends EventEmitter {
  private pc: RTCPeerConnection;
  private dataChannel: RTCDataChannel | null = null;
  private reliableChannel: RTCDataChannel | null = null;
  private peerId: string;
  private isInitiator: boolean;
  private transfers: Map<string, FileTransfer> = new Map();
  private chunkSize = 16384; // 16KB chunks

  constructor(peerId: string, isInitiator: boolean, iceServers?: RTCIceServer[]) {
    super();
    this.peerId = peerId;
    this.isInitiator = isInitiator;

    // Initialize peer connection with ICE servers
    this.pc = new RTCPeerConnection({
      iceServers: iceServers || [
        { urls: 'stun:stun.l.google.com:19302' },
        { urls: 'stun:stun1.l.google.com:19302' },
      ],
    });

    this.setupEventHandlers();
    
    if (isInitiator) {
      this.createDataChannels();
    }
  }

  private setupEventHandlers() {
    // ICE candidate events
    this.pc.onicecandidate = (event) => {
      if (event.candidate) {
        this.emit('icecandidate', event.candidate);
      }
    };

    // Connection state changes
    this.pc.onconnectionstatechange = () => {
      this.emit('connectionstatechange', this.pc.connectionState);
      
      if (this.pc.connectionState === 'connected') {
        this.emit('connected');
      } else if (this.pc.connectionState === 'failed' || this.pc.connectionState === 'closed') {
        this.emit('disconnected');
      }
    };

    // Data channel events (for receiver)
    this.pc.ondatachannel = (event) => {
      const channel = event.channel;
      
      if (channel.label === 'files') {
        this.dataChannel = channel;
        this.setupDataChannel(this.dataChannel);
      } else if (channel.label === 'reliable') {
        this.reliableChannel = channel;
        this.setupReliableChannel(this.reliableChannel);
      }
    };
  }

  private createDataChannels() {
    // Main data channel for file transfers
    this.dataChannel = this.pc.createDataChannel('files', {
      ordered: false,
      maxRetransmits: 0,
    });
    this.setupDataChannel(this.dataChannel);

    // Reliable channel for metadata and control messages
    this.reliableChannel = this.pc.createDataChannel('reliable', {
      ordered: true,
    });
    this.setupReliableChannel(this.reliableChannel);
  }

  private setupDataChannel(channel: RTCDataChannel) {
    channel.binaryType = 'arraybuffer';
    
    channel.onopen = () => {
      this.emit('datachannel:open');
    };

    channel.onclose = () => {
      this.emit('datachannel:close');
    };

    channel.onerror = (error) => {
      this.emit('datachannel:error', error);
    };

    channel.onmessage = (event) => {
      this.handleDataMessage(event.data);
    };
  }

  private setupReliableChannel(channel: RTCDataChannel) {
    channel.onopen = () => {
      this.emit('reliable:open');
    };

    channel.onmessage = (event) => {
      this.handleControlMessage(event.data);
    };
  }

  private handleControlMessage(data: any) {
    try {
      const message = JSON.parse(data);
      
      switch (message.type) {
        case 'file:metadata':
          this.handleIncomingFileMetadata(message.metadata);
          break;
        case 'file:request':
          this.handleFileRequest(message.fileId);
          break;
        case 'file:chunk:ack':
          this.handleChunkAck(message.fileId, message.chunkIndex);
          break;
        case 'file:complete':
          this.handleFileComplete(message.fileId);
          break;
        case 'file:error':
          this.handleFileError(message.fileId, message.error);
          break;
        case 'chat':
          this.emit('chat', message.text, message.from);
          break;
      }
    } catch (error) {
      console.error('Error handling control message:', error);
    }
  }

  private handleDataMessage(data: ArrayBuffer) {
    // First 36 bytes are file ID (UUID), next 4 bytes are chunk index
    const header = new DataView(data, 0, 40);
    const fileIdBytes = new Uint8Array(data, 0, 36);
    const fileId = new TextDecoder().decode(fileIdBytes);
    const chunkIndex = header.getUint32(36, true);
    const chunkData = data.slice(40);

    const transfer = this.transfers.get(fileId);
    if (!transfer) {
      console.error('Received chunk for unknown file:', fileId);
      return;
    }

    transfer.chunks[chunkIndex] = chunkData;
    transfer.progress = (transfer.chunks.filter(c => c).length / transfer.metadata.chunks) * 100;
    
    this.emit('file:progress', fileId, transfer.progress);

    // Send acknowledgment
    this.sendControlMessage({
      type: 'file:chunk:ack',
      fileId,
      chunkIndex,
    });

    // Check if transfer is complete
    if (transfer.chunks.filter(c => c).length === transfer.metadata.chunks) {
      this.completeFileTransfer(fileId);
    }
  }

  private handleIncomingFileMetadata(metadata: FileMetadata) {
    const transfer: FileTransfer = {
      metadata,
      progress: 0,
      status: 'pending',
      chunks: new Array(metadata.chunks),
    };

    this.transfers.set(metadata.id, transfer);
    this.emit('file:incoming', metadata);
  }

  private handleFileRequest(fileId: string) {
    this.emit('file:requested', fileId);
  }

  private handleChunkAck(fileId: string, chunkIndex: number) {
    // Continue sending next chunk if needed
    const transfer = this.transfers.get(fileId);
    if (transfer && transfer.status === 'transferring') {
      this.sendNextChunk(fileId, chunkIndex + 1);
    }
  }

  private handleFileComplete(fileId: string) {
    const transfer = this.transfers.get(fileId);
    if (transfer) {
      transfer.status = 'completed';
      this.emit('file:completed', fileId);
    }
  }

  private handleFileError(fileId: string, error: string) {
    const transfer = this.transfers.get(fileId);
    if (transfer) {
      transfer.status = 'failed';
      this.emit('file:error', fileId, error);
    }
  }

  private completeFileTransfer(fileId: string) {
    const transfer = this.transfers.get(fileId);
    if (!transfer) return;

    // Combine all chunks into a single blob
    const blob = new Blob(transfer.chunks, { type: transfer.metadata.type });
    transfer.status = 'completed';
    
    this.emit('file:received', fileId, blob, transfer.metadata);

    // Notify sender that transfer is complete
    this.sendControlMessage({
      type: 'file:complete',
      fileId,
    });
  }

  async createOffer(): Promise<RTCSessionDescriptionInit> {
    const offer = await this.pc.createOffer();
    await this.pc.setLocalDescription(offer);
    return offer;
  }

  async createAnswer(offer: RTCSessionDescriptionInit): Promise<RTCSessionDescriptionInit> {
    await this.pc.setRemoteDescription(offer);
    const answer = await this.pc.createAnswer();
    await this.pc.setLocalDescription(answer);
    return answer;
  }

  async handleAnswer(answer: RTCSessionDescriptionInit) {
    await this.pc.setRemoteDescription(answer);
  }

  async addIceCandidate(candidate: RTCIceCandidateInit) {
    try {
      await this.pc.addIceCandidate(candidate);
    } catch (error) {
      console.error('Error adding ICE candidate:', error);
    }
  }

  async sendFile(file: File): Promise<string> {
    const fileId = crypto.randomUUID();
    const chunks = Math.ceil(file.size / this.chunkSize);
    
    const metadata: FileMetadata = {
      id: fileId,
      name: file.name,
      size: file.size,
      type: file.type,
      chunks,
    };

    // Store transfer info
    const transfer: FileTransfer = {
      metadata,
      progress: 0,
      status: 'pending',
      chunks: [],
    };
    this.transfers.set(fileId, transfer);

    // Send metadata first
    this.sendControlMessage({
      type: 'file:metadata',
      metadata,
    });

    // Start sending chunks
    transfer.status = 'transferring';
    this.sendFileChunks(file, fileId);

    return fileId;
  }

  private async sendFileChunks(file: File, fileId: string) {
    const transfer = this.transfers.get(fileId);
    if (!transfer) return;

    for (let i = 0; i < transfer.metadata.chunks; i++) {
      const start = i * this.chunkSize;
      const end = Math.min(start + this.chunkSize, file.size);
      const chunk = await file.slice(start, end).arrayBuffer();
      
      // Create header with file ID and chunk index
      const header = new ArrayBuffer(40);
      const headerView = new DataView(header);
      const encoder = new TextEncoder();
      const fileIdBytes = encoder.encode(fileId);
      new Uint8Array(header, 0, 36).set(fileIdBytes);
      headerView.setUint32(36, i, true);
      
      // Combine header and chunk data
      const message = new Uint8Array(header.byteLength + chunk.byteLength);
      message.set(new Uint8Array(header), 0);
      message.set(new Uint8Array(chunk), header.byteLength);
      
      // Send via data channel with flow control
      await this.sendWithFlowControl(message.buffer);
      
      transfer.progress = ((i + 1) / transfer.metadata.chunks) * 100;
      this.emit('file:progress', fileId, transfer.progress);
    }
  }

  private async sendWithFlowControl(data: ArrayBuffer): Promise<void> {
    if (!this.dataChannel || this.dataChannel.readyState !== 'open') {
      throw new Error('Data channel not ready');
    }

    // Check buffered amount and wait if necessary
    while (this.dataChannel.bufferedAmount > 65536) { // 64KB threshold
      await new Promise(resolve => setTimeout(resolve, 50));
    }

    this.dataChannel.send(data);
  }

  private sendNextChunk(fileId: string, chunkIndex: number) {
    // Implementation depends on having access to the original file
    // This would be called after receiving an ACK for the previous chunk
  }

  private sendControlMessage(message: any) {
    if (!this.reliableChannel || this.reliableChannel.readyState !== 'open') {
      console.error('Reliable channel not ready');
      return;
    }

    this.reliableChannel.send(JSON.stringify(message));
  }

  sendChatMessage(text: string, from: string) {
    this.sendControlMessage({
      type: 'chat',
      text,
      from,
      timestamp: Date.now(),
    });
  }

  acceptFile(fileId: string) {
    this.sendControlMessage({
      type: 'file:request',
      fileId,
    });
  }

  getConnectionState(): RTCPeerConnectionState {
    return this.pc.connectionState;
  }

  close() {
    if (this.dataChannel) {
      this.dataChannel.close();
    }
    if (this.reliableChannel) {
      this.reliableChannel.close();
    }
    this.pc.close();
    this.removeAllListeners();
  }
}