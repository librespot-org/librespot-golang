package crypto

/* Shannon: Shannon stream cipher and MAC -- reference implementation */

/*
THIS SOFTWARE IS PROVIDED ``AS IS'' AND ANY EXPRESS OR IMPLIED
WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF
MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE AND AGAINST
INFRINGEMENT ARE DISCLAIMED.  IN NO EVENT SHALL THE AUTHOR OR
CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL,
EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO,
PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR
PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF
LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING
NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

/* interface */

// #include <stdlib.h>
// #include <string.h>
/* $Id: shn.h 182 2009-03-12 08:21:53Z zagor $ */
/* Shannon: Shannon stream cipher and MAC header files */

/*
THIS SOFTWARE IS PROVIDED ``AS IS'' AND ANY EXPRESS OR IMPLIED
WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF
MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE AND AGAINST
INFRINGEMENT ARE DISCLAIMED.  IN NO EVENT SHALL THE AUTHOR OR
CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL,
EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO,
PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR
PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF
LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING
NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

type shn_ctx struct {
	R     [N]uint32
	CRC   [N]uint32
	initR [N]uint32
	konst uint32
	sbuf  uint32
	mbuf  uint32
	nbuf  int
}

/* interface definitions */
/*
 * FOLD is how many register cycles need to be performed after combining the
 * last byte of key and non-linear feedback, before every byte depends on every
 * byte of the key. This depends on the feedback and nonlinear functions, and
 * on where they are combined into the register. Making it same as the
 * register length is a safe and conservative choice.
 */

const N int = 16

const FOLD int = 16

const initkonst uint32 = 0x6996c53a

const KEYP int = 13

/* some useful macros -- machine independent little-endian */

func toByte(x uint32, i int) uint8 {
	return uint8((x >> uint(8*i)) & 0xFF)
}

func rotl(w uint32, x int) uint32 {
	return w<<uint(x) | (w&0xffffffff)>>uint(32-x)
}

func byte2word(b []byte) uint32 {
	return (uint32(b[3])&0xFF)<<24 | (uint32(b[2])&0xFF)<<16 | (uint32(b[1])&0xFF)<<8 | (uint32(b[0]) & 0xFF)
}

func word2byte(w uint32, b []byte) {
	b[3] = byte(toByte(w, 3))
	b[2] = byte(toByte(w, 2))
	b[1] = byte(toByte(w, 1))
	b[0] = byte(toByte(w, 0))
}

func xorword(w uint32, b []byte) {
	b[3] ^= byte(toByte(w, 3))
	b[2] ^= byte(toByte(w, 2))
	b[1] ^= byte(toByte(w, 1))
	b[0] ^= byte(toByte(w, 0))
}

/* Nonlinear transform (sbox) of a word.
 * There are two slightly different combinations.
 */
func sbox1(w uint32) uint32 {
	w ^= rotl(w, 5) | rotl(w, 7)
	w ^= rotl(w, 19) | rotl(w, 22)
	return w
}

func sbox2(w uint32) uint32 {
	w ^= rotl(w, 7) | rotl(w, 22)
	w ^= rotl(w, 5) | rotl(w, 19)
	return w
}

/* cycle the contents of the register and calculate output word in c->sbuf.
 */
func cycle(c *shn_ctx) {
	var t uint32
	var i int

	/* nonlinear feedback function */
	t = c.R[12] ^ c.R[13] ^ c.konst

	t = sbox1(t) ^ rotl(c.R[0], 1)

	/* shift register */
	for i = 1; i < N; i++ {
		c.R[i-1] = c.R[i]
	}
	c.R[N-1] = t
	t = sbox2(c.R[2] ^ c.R[15])
	c.R[0] ^= t
	c.sbuf = t ^ c.R[8] ^ c.R[12]
}

/* The Shannon MAC function is modelled after the concepts of Phelix and SHA.
 * Basically, words to be accumulated in the MAC are incorporated in t`wo
 * different ways:
 * 1. They are incorporated into the stream cipher register at a place
 *    where they will immediately have a nonlinear effect on the state
 * 2. They are incorporated into bit-parallel CRC-16 registers; the
 *    contents of these registers will be used in MAC finalization.
 */

/* Accumulate a CRC of input words, later to be fed into MAC.
 * This is actually 32 parallel CRC-16s, using the IBM CRC-16
 * polynomial x^16 + x^15 + x^2 + 1.
 */
func crcfunc(c *shn_ctx, i uint32) {
	var t uint32
	var j int

	/* Accumulate CRC of input */
	t = c.CRC[0] ^ c.CRC[2] ^ c.CRC[15] ^ i

	for j = 1; j < N; j++ {
		c.CRC[j-1] = c.CRC[j]
	}
	c.CRC[N-1] = t
}

/* Normal MAC word processing: do both stream register and CRC.
 */
func macfunc(c *shn_ctx, i uint32) {
	crcfunc(c, i)
	c.R[KEYP] ^= i
}

/* initialise to known state
 */
func shn_initstate(c *shn_ctx) {
	var i int

	/* Register initialised to Fibonacci numbers; Counter zeroed. */
	c.R[0] = 1

	c.R[1] = 1
	for i = 2; i < N; i++ {
		c.R[i] = c.R[i-1] + c.R[i-2]
	}
	c.konst = initkonst
}

/* Save the current register state
 */
func shn_savestate(c *shn_ctx) {
	var i int

	for i = 0; i < N; i++ {
		c.initR[i] = c.R[i]
	}
}

/* initialise to previously saved register state
 */
func shn_reloadstate(c *shn_ctx) {
	var i int

	for i = 0; i < N; i++ {
		c.R[i] = c.initR[i]
	}
}

/* Initialise "konst"
 */
func shn_genkonst(c *shn_ctx) {
	c.konst = c.R[0]
}

/* Load key material into the register
 */
// #define addkey(k) \
// 	c->R[KEYP] ^= (k);
func addkey(c *shn_ctx, k uint32) {
	c.R[KEYP] ^= k
}

/* extra nonlinear diffusion of register for key and MAC */
func shn_diffuse(c *shn_ctx) {
	var i int

	for i = 0; i < FOLD; i++ {
		cycle(c)
	}
}

/* Common actions for loading key material
 * Allow non-word-multiple key and nonce material.
 * Note also initializes the CRC register as a side effect.
 */
func shn_loadkey(c *shn_ctx, key []byte, keylen int) {
	var i int
	var j int
	var k uint32
	var xtra [4]uint8

	/* start folding in key */
	for i = 0; i < keylen&^0x3; i += 4 {
		k = byte2word(key[i:])
		addkey(c, k)
		cycle(c)
	}

	/* if there were any extra key bytes, zero pad to a word */
	if i < keylen {
		for j = 0; i < keylen; i++ { /* i unchanged */
			xtra[j] = uint8(key[i])
			j++ /* j unchanged */
		}
		for ; j < 4; j++ {
			xtra[j] = 0
		}
		k = byte2word(xtra[:])
		addkey(c, k)
		cycle(c)
	}

	/* also fold in the length of the key */
	addkey(c, uint32(keylen))

	cycle(c)

	/* save a copy of the register */
	for i = 0; i < N; i++ {
		c.CRC[i] = c.R[i]
	}

	/* now diffuse */
	shn_diffuse(c)

	/* now xor the copy back -- makes key loading irreversible */
	for i = 0; i < N; i++ {
		c.R[i] ^= c.CRC[i]
	}
}

/* Published "key" interface
 */
func shn_key(c *shn_ctx, key []byte, keylen int) {
	shn_initstate(c)
	shn_loadkey(c, key, keylen)
	shn_genkonst(c) /* in case we proceed to stream generation */
	shn_savestate(c)
	c.nbuf = 0
}

/* Published "IV" interface
 */
func shn_nonce(c *shn_ctx, nonce []byte, noncelen int) {
	shn_reloadstate(c)
	c.konst = initkonst
	shn_loadkey(c, nonce, noncelen)
	shn_genkonst(c)
	c.nbuf = 0
}

/* XOR pseudo-random bytes into buffer
 * Note: doesn't play well with MAC functions.
 */
func shn_stream(c *shn_ctx, buf []byte, nbytes int) {
	var endbuf []byte

	/* Handle any previously buffered bytes */
	for c.nbuf != 0 && nbytes != 0 {
		buf[0] ^= byte(c.sbuf & 0xFF)
		buf = buf[1:]
		c.sbuf >>= 8
		c.nbuf -= 8
		nbytes--
	}

	/* Handle whole words */
	endbuf = buf[uint32(nbytes)&^(uint32(0x03)):]

	for -cap(buf) < -cap(endbuf) {
		cycle(c)
		xorword(c.sbuf, buf)
		buf = buf[4:]
	}

	/* Handle any trailing bytes */
	nbytes &= 0x03

	if nbytes != 0 {
		cycle(c)
		c.nbuf = 32
		for c.nbuf != 0 && nbytes != 0 {
			buf[0] ^= byte(c.sbuf & 0xFF)
			buf = buf[1:]
			c.sbuf >>= 8
			c.nbuf -= 8
			nbytes--
		}
	}
}

/* accumulate words into MAC without encryption
 * Note that plaintext is accumulated for MAC.
 */
func shn_maconly(c *shn_ctx, buf []byte, nbytes int) {
	var endbuf []byte

	/* Handle any previously buffered bytes */
	if c.nbuf != 0 {
		for c.nbuf != 0 && nbytes != 0 {
			c.mbuf ^= uint32(buf[0]) << uint(32-c.nbuf)
			buf = buf[1:]
			c.nbuf -= 8
			nbytes--
		}

		if c.nbuf != 0 { /* not a whole word yet */
			return
		}

		/* LFSR already cycled */
		macfunc(c, c.mbuf)
	}

	/* Handle whole words */
	endbuf = buf[uint32(nbytes)&^(uint32(0x03)):]

	for -cap(buf) < -cap(endbuf) {
		cycle(c)
		macfunc(c, byte2word(buf))
		buf = buf[4:]
	}

	/* Handle any trailing bytes */
	nbytes &= 0x03

	if nbytes != 0 {
		cycle(c)
		c.mbuf = 0
		c.nbuf = 32
		for c.nbuf != 0 && nbytes != 0 {
			c.mbuf ^= uint32(buf[0]) << uint(32-c.nbuf)
			buf = buf[1:]
			c.nbuf -= 8
			nbytes--
		}
	}
}

/* Combined MAC and encryption.
 * Note that plaintext is accumulated for MAC.
 */
func shn_encrypt(c *shn_ctx, buf []byte, nbytes int) {
	var endbuf []byte
	var t uint32 = 0

	/* Handle any previously buffered bytes */
	if c.nbuf != 0 {
		for c.nbuf != 0 && nbytes != 0 {
			c.mbuf ^= uint32(buf[0]) << uint(32-c.nbuf)
			buf[0] ^= byte((c.sbuf >> uint(32-c.nbuf)) & 0xFF)
			buf = buf[1:]
			c.nbuf -= 8
			nbytes--
		}

		if c.nbuf != 0 { /* not a whole word yet */
			return
		}

		/* LFSR already cycled */
		macfunc(c, c.mbuf)
	}

	/* Handle whole words */
	endbuf = buf[uint32(nbytes)&^(uint32(0x03)):]

	for -cap(buf) < -cap(endbuf) {
		cycle(c)
		t = byte2word(buf)
		macfunc(c, t)
		t ^= c.sbuf
		word2byte(t, buf)
		buf = buf[4:]
	}

	/* Handle any trailing bytes */
	nbytes &= 0x03

	if nbytes != 0 {
		cycle(c)
		c.mbuf = 0
		c.nbuf = 32
		for c.nbuf != 0 && nbytes != 0 {
			c.mbuf ^= uint32(buf[0]) << uint(32-c.nbuf)
			buf[0] ^= byte((c.sbuf >> uint(32-c.nbuf)) & 0xFF)
			buf = buf[1:]
			c.nbuf -= 8
			nbytes--
		}
	}
}

/* Combined MAC and decryption.
 * Note that plaintext is accumulated for MAC.
 */
func shn_decrypt(c *shn_ctx, buf []byte, nbytes int) {
	var endbuf []byte
	var t uint32 = 0

	/* Handle any previously buffered bytes */
	if c.nbuf != 0 {
		for c.nbuf != 0 && nbytes != 0 {
			buf[0] ^= byte((c.sbuf >> uint(32-c.nbuf)) & 0xFF)
			c.mbuf ^= uint32(buf[0]) << uint(32-c.nbuf)
			buf = buf[1:]
			c.nbuf -= 8
			nbytes--
		}

		if c.nbuf != 0 { /* not a whole word yet */
			return
		}

		/* LFSR already cycled */
		macfunc(c, c.mbuf)
	}

	/* Handle whole words */
	endbuf = buf[uint32(nbytes)&^(uint32(0x03)):]

	for -cap(buf) < -cap(endbuf) {
		cycle(c)
		t = byte2word(buf) ^ c.sbuf
		macfunc(c, t)
		word2byte(t, buf)
		buf = buf[4:]
	}

	/* Handle any trailing bytes */
	nbytes &= 0x03

	if nbytes != 0 {
		cycle(c)
		c.mbuf = 0
		c.nbuf = 32
		for c.nbuf != 0 && nbytes != 0 {
			buf[0] ^= byte((c.sbuf >> uint(32-c.nbuf)) & 0xFF)
			c.mbuf ^= uint32(buf[0]) << uint(32-c.nbuf)
			buf = buf[1:]
			c.nbuf -= 8
			nbytes--
		}
	}
}

/* Having accumulated a MAC, finish processing and return it.
 * Note that any unprocessed bytes are treated as if
 * they were encrypted zero bytes, so plaintext (zero) is accumulated.
 */
func shn_finish(c *shn_ctx, buf []byte, nbytes int) {
	var i int

	/* Handle any previously buffered bytes */
	if c.nbuf != 0 {
		/* LFSR already cycled */
		macfunc(c, c.mbuf)
	}

	/* perturb the MAC to mark end of input.
	 * Note that only the stream register is updated, not the CRC. This is an
	 * action that can't be duplicated by passing in plaintext, hence
	 * defeating any kind of extension attack.
	 */
	cycle(c)

	addkey(c, initkonst^(uint32(c.nbuf)<<3))
	c.nbuf = 0

	/* now add the CRC to the stream register and diffuse it */
	for i = 0; i < N; i++ {
		c.R[i] ^= c.CRC[i]
	}
	shn_diffuse(c)

	/* produce output from the stream buffer */
	for nbytes > 0 {
		cycle(c)
		if nbytes >= 4 {
			word2byte(c.sbuf, buf)
			nbytes -= 4
			buf = buf[4:]
		} else {
			for i = 0; i < nbytes; i++ {
				buf[i] = byte(toByte(c.sbuf, i))
			}
			break
		}
	}
}
