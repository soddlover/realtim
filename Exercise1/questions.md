Exercise 1 - Theory questions
-----------------------------

### Concepts

What is the difference between *concurrency* and *parallelism*?
> Concurrency er som at en person sjonglerer, parallelism er som at flere personer sjonglerer. Concurrency håndterer flere oppgaver på en gang, men utfører dem sekvensielt, parallelism utfører i tillegg flere oppgaver samtidig.

What is the difference between a *race condition* and a *data race*? 
> Race conditions er et bredt beskrivende konsept som handler om korrektheten til et program basert på timingen/sekvensen prosesser blir utført i. Data race er en spesifikk type race condition hvor to prosesser prøver å modifisere/lese fra samme minne samtidig.
 
*Very* roughly - what does a *scheduler* do, and how does it do it?
> Scheduleren sier hvilken tråd som skal brukes basert på timing.


### Engineering

Why would we use multiple threads? What kinds of problems do threads solve?
> Med flere tråder kan man separere forskjellige oppgaver. På den måten kan man utføre flere operasjoner samtidig/få det til å se ut som flere operasjoner blir utført samtidig. Man kan separere uavhengige ting. Dette øker lesbarheten av kode.

Some languages support "fibers" (sometimes called "green threads") or "coroutines"? What are they, and why would we rather use them over threads?
> Fibers er tråder som kontrolleres av applikasjonen, og ikke OS-scheduleren. Coroutines er subrutiner som kan stoppes og fortsettes, styrt av applikasjonen. Dette er fordelaktig når det er enkle/lette oppgaver som skal utføres.

Does creating concurrent programs make the programmer's life easier? Harder? Maybe both?
> Det gjør livet enklere med tanke på separasjon og lesbarhet, men det blir vanskeligere med tanke på race conditions som må vurderes i tillegg til bugs kan være vanskelige å reprodusere, finne og squashe.

What do you think is best - *shared variables* or *message passing*?
> Message passing fordi shared variables er dårlig stemning. Prosesser er egoistiske og bryr seg ikke om hva andre prosesser vil med samme variabel.


