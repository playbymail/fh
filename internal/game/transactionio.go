package game

// Port of transactionio.c. Reads and writes the binary file
// "interspecies.dat".
//
// NOTE: the C file defines a file-local struct confusingly named
// binary_ship_data_t for the transaction record. That local layout is
// what is ported here, not the ship record from data.h.
//
// Skipped (JSON/S-expression exporters): transactionDataAsJson,
// transactionDataAsSExpr.

import (
	"encoding/binary"
	"fmt"
	"os"
)

// binary_transaction_data is 148 bytes:
//
//	offset   0: int32 type
//	offset   4: int16 donor
//	offset   6: int16 recipient
//	offset   8: int32 value
//	offset  12: uint8 x, y, z, pn
//	offset  16: int32 number1
//	offset  20: uint8 name1[40]
//	offset  60: int32 number2
//	offset  64: uint8 name2[40]
//	offset 104: int32 number3
//	offset 108: uint8 name3[40]
const binary_transaction_data_size = 148

/* Read transactions from file. */
func get_transaction_data() {
	/* Get size of file. */
	sb, err := os.Stat("interspecies.dat")
	if err != nil {
		num_transactions = 0
		return
	}

	// get number of records in the file
	num_transactions = int(sb.Size()) / binary_transaction_data_size
	if sb.Size() != int64(num_transactions*binary_transaction_data_size) {
		fmt.Fprintf(os.Stderr, "\nFile interspecies.dat contains extra bytes (%d > %d)!\n\n",
			sb.Size(), num_transactions*binary_transaction_data_size)
		os.Exit(255)
	} else if num_transactions == 0 {
		// nothing to do
		return
	} else if num_transactions > MAX_TRANSACTIONS {
		fmt.Fprintf(os.Stderr, "\nFile interspecies.dat contains too many records (%d > %d)!\n\n", num_transactions,
			MAX_TRANSACTIONS)
		os.Exit(255)
	}

	/* Open transactions file and read it all into memory. */
	binData, err := os.ReadFile("interspecies.dat")
	if err != nil {
		fmt.Fprintf(os.Stderr, "get_transaction_data: %v\n", err)
		fmt.Fprintf(os.Stderr, "\nCannot open file 'interspecies.dat' for reading!\n\n")
		os.Exit(255)
	}
	if len(binData) < num_transactions*binary_transaction_data_size {
		fmt.Fprintf(os.Stderr, "\nCannot read file 'interspecies.dat' into memory!\n")
		fmt.Fprintf(os.Stderr, "\n\tattempted to read %d transaction entries\n\n", num_transactions)
		os.Exit(255)
	}

	/* translate data */
	for i := 0; i < num_transactions; i++ {
		rec := binData[i*binary_transaction_data_size:]
		transaction[i].trans_type = int(int32(binary.LittleEndian.Uint32(rec[0:])))
		transaction[i].donor = int(int16(binary.LittleEndian.Uint16(rec[4:])))
		transaction[i].recipient = int(int16(binary.LittleEndian.Uint16(rec[6:])))
		transaction[i].value = int(int32(binary.LittleEndian.Uint32(rec[8:])))
		transaction[i].x = int(rec[12])
		transaction[i].y = int(rec[13])
		transaction[i].z = int(rec[14])
		transaction[i].pn = int(rec[15])
		transaction[i].number1 = int(int32(binary.LittleEndian.Uint32(rec[16:])))
		transaction[i].number2 = int(int32(binary.LittleEndian.Uint32(rec[60:])))
		transaction[i].number3 = int(int32(binary.LittleEndian.Uint32(rec[104:])))
		transaction[i].name1 = nameString(rec[20:60])
		transaction[i].name2 = nameString(rec[64:104])
		transaction[i].name3 = nameString(rec[108:148])
	}
}

func save_transaction_data() {
	/* Open file 'interspecies.dat' for writing. */
	fp, err := os.Create("interspecies.dat")
	if err != nil {
		fmt.Fprintf(os.Stderr, "save_transaction_data: %v\n", err)
		fmt.Fprintf(os.Stderr, "\n\tCannot create file 'interspecies.dat'!\n\n")
		os.Exit(255)
	}

	if num_transactions > 0 {
		binData := make([]byte, num_transactions*binary_transaction_data_size)

		/* translate data */
		for i := 0; i < num_transactions; i++ {
			rec := binData[i*binary_transaction_data_size:]
			binary.LittleEndian.PutUint32(rec[0:], uint32(int32(transaction[i].trans_type)))
			binary.LittleEndian.PutUint16(rec[4:], uint16(int16(transaction[i].donor)))
			binary.LittleEndian.PutUint16(rec[6:], uint16(int16(transaction[i].recipient)))
			binary.LittleEndian.PutUint32(rec[8:], uint32(int32(transaction[i].value)))
			rec[12] = byte(transaction[i].x)
			rec[13] = byte(transaction[i].y)
			rec[14] = byte(transaction[i].z)
			rec[15] = byte(transaction[i].pn)
			binary.LittleEndian.PutUint32(rec[16:], uint32(int32(transaction[i].number1)))
			binary.LittleEndian.PutUint32(rec[60:], uint32(int32(transaction[i].number2)))
			binary.LittleEndian.PutUint32(rec[104:], uint32(int32(transaction[i].number3)))
			copyName(rec[20:60], transaction[i].name1)
			copyName(rec[64:104], transaction[i].name2)
			copyName(rec[108:148], transaction[i].name3)
		}

		/* Write array to disk. */
		if _, err := fp.Write(binData); err != nil {
			fmt.Fprintf(os.Stderr, "save_transaction_data: %v\n", err)
			fmt.Fprintf(os.Stderr, "\n\n\tCannot write to 'interspecies.dat'!\n\n")
			os.Exit(255)
		}
	}
	fp.Close()
}
