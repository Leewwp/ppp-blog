plugins {
    java
    id("io.freefair.lombok") version "8.13"
    id("run.halo.plugin.devtools") version "0.6.1"
}

val haloVersion = providers.gradleProperty("haloVersion")
    .orElse("2.22.0")
    .get()
val haloRuntimeVersion = providers.gradleProperty("haloRuntimeVersion")
    .orElse("2.22")
    .get()

group = "com.ppp.plugin"
version = "1.0.0"

repositories {
    maven { url = uri("https://maven.aliyun.com/repository/public/") }
    maven { url = uri("https://maven.aliyun.com/repository/gradle-plugin/") }
    mavenCentral()
}

dependencies {
    implementation(platform("run.halo.tools.platform:plugin:$haloVersion"))

    compileOnly("run.halo.app:api:$haloVersion")

    implementation("org.springframework.boot:spring-boot-starter-webflux")

    compileOnly("org.projectlombok:lombok")
    annotationProcessor("org.projectlombok:lombok")

    testImplementation("run.halo.app:api:$haloVersion")
    testImplementation("org.springframework.boot:spring-boot-starter-test")
    testRuntimeOnly("org.junit.platform:junit-platform-launcher")
}

java {
    toolchain {
        languageVersion = JavaLanguageVersion.of(21)
    }
}

tasks.withType<JavaCompile>().configureEach {
    options.encoding = "UTF-8"
    options.release = 21
}

tasks.withType<Test>().configureEach {
    useJUnitPlatform()
}

halo {
    version = haloRuntimeVersion
}
